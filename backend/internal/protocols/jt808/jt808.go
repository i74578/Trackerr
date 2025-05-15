package jt808

import (
	"banjo.dev/trackerr/internal/database"
	"banjo.dev/trackerr/internal/model"
	"banjo.dev/trackerr/internal/utils"
	"bytes"
	b64 "encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

// JT808 specific constants
const (
	StartByte                   byte          = 0x7E
	EndByte                     byte          = 0x7E
	CoordinatePrecision         float64       = 1000000
	MsgTypeTermUniversalRes     uint16        = 0x0001
	MsgTypeHeartbeat            uint16        = 0x0002
	MsgTypeLogout               uint16        = 0x0003
	MsgTypeRegistrion           uint16        = 0x0100
	MsgTypeAuth                 uint16        = 0x0102
	MsgTypeLocation             uint16        = 0x0200
	MsgTypeVersionInfo          uint16        = 0x0205
	MsgTypeLocationBatch        uint16        = 0x0704
	MsgTypeUpstreamData         uint16        = 0x0900
	MsgTypeCmdRes               uint16        = 0x6006
	MsgTypePlatformUniversalRes uint16        = 0x8001
	MsgTypeTermRegistrationRes  uint16        = 0x8100
	MsgTypeVersionInfoRes       uint16        = 0x8205
	MsgTypeCmdSend              uint16        = 0x8300
	ResultSuccess               uint8         = 0x00
	ResultFailure               uint8         = 0x01
	ResultIncorrectInformation  uint8         = 0x02
	NotSupporting               uint8         = 0x03
	AlarmProcessingConfirmation uint8         = 0x04
	HeartbeatInterval           time.Duration = 5 * time.Minute
)

// Perform authentication and registration
func PerformAuth(conn net.Conn, p model.Packet) (string, error) {
	// Abort if first packet is not registration nor authentication
	if p.PacketType != MsgTypeRegistrion && p.PacketType != MsgTypeAuth {
		return "", fmt.Errorf("Expected registion or authentication request but received:", p.PacketType)
	}

	buf := bytes.NewBuffer([]byte{})
	trackerID := p.DeviceID
	authcode := make([]byte, 12)
	if p.PacketType == MsgTypeRegistrion {
		// Generate auth  code
		rand.Read(authcode)
		// Write start of response message
		buf = bytes.NewBuffer([]byte{})
		binary.Write(buf, binary.BigEndian, p.SerialNumber)

		// Save generated authcode to database
		encodedCode := b64.StdEncoding.EncodeToString(authcode)
		err := database.SaveAuthCode(model.AuthCode{TrackerId: trackerID, Code: encodedCode})
		if err != nil {
			// Create and send failed registration response
			buf.WriteByte(2)
			//buf.Write(authcode)

			SendMsg(conn, MsgTypeTermRegistrationRes, buf.Bytes(), 0, trackerID)
			return "", fmt.Errorf("Failed to store auth code in database. This may be because tracker %v it is not registered\n", trackerID)
		}

		// Create and send registration response
		buf.WriteByte(0)
		buf.Write(authcode)

		SendMsg(conn, MsgTypeTermRegistrationRes, buf.Bytes(), 0, trackerID)

		// Try to read authentication message
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		utils.ReadBytes(conn, 1) // skip start byte
		p, err = ParseMsg(conn, utils.NullTime{time.Duration(60), true})
		if err != nil {
			return "", fmt.Errorf("Failed to parse authentication request")
		}
		if p.PacketType != MsgTypeAuth {
			return "", fmt.Errorf("Expected authentiction request packet but received:", p.PacketType)
		}
	} else {
		// Received authentication message
		// Read authcode from database
		ac, err := database.FetchAuthCode(trackerID)
		// Send response indicating incorrect information
		if err != nil {
			SendUniversalRes(conn, MsgTypeAuth, p.SerialNumber, ResultIncorrectInformation, trackerID)
			return "", fmt.Errorf("Failed to fetch auth code for %v: %v", trackerID, err)
		}
		authcode, _ = b64.StdEncoding.DecodeString(ac.Code)
	}

	// Indicate failure if authcode does not match
	if !bytes.Equal(authcode, p.Payload) {
		log.Println("Recevied wrong auth code")
		SendUniversalRes(conn, MsgTypeAuth, p.SerialNumber, ResultFailure, trackerID)
		return "", fmt.Errorf("Recevied wrong auth code")
	}
	log.Println("Received valid authcode from device")
	SendUniversalRes(conn, MsgTypeAuth, p.SerialNumber, ResultSuccess, trackerID)
	return trackerID, nil
}

// Parse JT808 message
func ParseMsg(conn net.Conn, maxWait utils.NullTime) (model.Packet, error) {
	var p model.Packet
	var err error
	// Use read deadline if parameter is set
	if maxWait.IsSet {
		conn.SetReadDeadline(time.Now().Add(maxWait.Time))
	}
	header, err := readBytesAndEscape(conn, 12)
	if err != nil {
		return p, fmt.Errorf("Failed to read header: %v", err)
	}

	// Map message id as packet type
	p.PacketType = binary.BigEndian.Uint16(header[0:2])
	p.PayloadLength = binary.BigEndian.Uint16(header[2:4]) & 0b1111111111
	p.DeviceID = hex.EncodeToString(header[4:10])
	p.SerialNumber = binary.BigEndian.Uint16(header[10:12])

	// Read payload
	p.Payload, err = readBytesAndEscape(conn, int(p.PayloadLength))
	if err != nil {
		return p, fmt.Errorf("Failed to read payload: %v", err)
	}

	trailer, err := readBytesAndEscape(conn, 2)
	if err != nil {
		return p, fmt.Errorf("Failed to read trailer: %v", err)
	}
	errorCheck := trailer[0]

	// Put data for CRC in buffer
	buf := bytes.NewBuffer([]byte(""))
	buf.Write(header)
	buf.Write(p.Payload)

	// Validate error code
	xordata := buf.Bytes()
	if xorbytes(xordata) != errorCheck {
		return p, fmt.Errorf("Invalid error check code. %v results in %v which is not equal to %v", xordata, xorbytes(xordata), errorCheck)
	}

	// Valdite end flag, which must be 0x7e
	endflag := trailer[1]
	if endflag != EndByte {
		return p, fmt.Errorf("Invalid end byte")
	}
	return p, nil
}

// Parse location message
func ParseLocationMsg(payload []byte) model.Locationdata {
	locationBytes := payload[8:22]
	return model.Locationdata{
		Lat:     binary.BigEndian.Uint32(locationBytes[0:4]),
		Lon:     binary.BigEndian.Uint32(locationBytes[4:8]),
		Speed:   binary.BigEndian.Uint16(locationBytes[10:12]),
		Heading: binary.BigEndian.Uint16(locationBytes[12:14]),
	}
}

// Parse command response
func ParseCmdRes(payload []byte) string {
	return string(payload[7:])
}

// Send JT808 specific message
func SendMsg(conn net.Conn, msgtype uint16, payload []byte, serialnum uint16, trackerID string) {
	buf := bytes.NewBuffer([]byte{})
	// Write start byte
	buf.Write([]byte{0x7e})
	// Map msgtype to message id
	binary.Write(buf, binary.BigEndian, msgtype)
	// Write body attribute
	binary.Write(buf, binary.BigEndian, uint16(len(payload)))
	// Write trackerID - MUST BE MAPPED TO PARAMETER
	id, _ := hex.DecodeString(trackerID)
	buf.Write(id)
	//binary.Write(buf,binary.BigEndian,[]byte{0x01,0x63,0x70,0x57,0x75,0x06})
	// Write Serial number
	binary.Write(buf, binary.BigEndian, serialnum)
	// Write payload
	buf.Write(payload)
	// Write error check code
	buf.WriteByte(xorbytes(buf.Bytes()[1:]))
	// Encapsulate entire message except start and end bytes
	escapedRes := encapsulate(buf.Bytes()[1:])
	// Write end byte
	escapedRes = append(append([]byte{0x7e}, escapedRes...), 0x7e)
	// Send buffer to client
	conn.Write(escapedRes)
	return
}

// Send jt808 upstream command
func SendCmd(conn net.Conn, payload string, tid string, serialNumber uint16) {
	cmdbuf := bytes.NewBuffer([]byte{0x01})
	cmdbuf.Write([]byte(payload))
	SendMsg(conn, MsgTypeCmdSend, cmdbuf.Bytes(), serialNumber, tid)
}

// Send platform universal response
func SendUniversalRes(conn net.Conn, packetType uint16, serialNumber uint16, result uint8, tid string) {
	payload := bytes.NewBuffer([]byte{})
	binary.Write(payload, binary.BigEndian, serialNumber)
	binary.Write(payload, binary.BigEndian, packetType)
	payload.WriteByte(result)
	SendMsg(conn, MsgTypePlatformUniversalRes, payload.Bytes(), 0, tid)
}

// Wrapper of readBytes which includes the JT808 escape process
func readBytesAndEscape(r io.Reader, n int) ([]byte, error) {
	escapedData, err := utils.ReadBytes(r, n)
	if err != nil {
		return escapedData, err
	}
	restoredData := make([]byte, 0, len(escapedData))
	for i := 0; i < len(escapedData); i++ {
		if escapedData[i] == 0x7d {
			// Since escaping results in 1 byte being discarded, a new byte must be read
			additionalData, err := utils.ReadBytes(r, 1)
			if err != nil {
				return escapedData, err
			}
			escapedData = append(escapedData, additionalData...)
			switch i++; escapedData[i] {
			case 0x01:
				// Turn 0x7d01 info 0x7d
				restoredData = append(restoredData, 0x7d)
			case 0x02:
				// Turn 0x7d02 info 0x7e
				restoredData = append(restoredData, 0x7e)

			default:
				restoredData = append(restoredData, escapedData[i])
			}
		} else {
			restoredData = append(restoredData, escapedData[i])
		}
	}
	return restoredData, err
}

func encapsulate(data []byte) []byte {
	escapedData := make([]byte, 0, len(data)*2)
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case 0x7d:
			// Turn 0x7d into 0x7d01
			escapedData = append(escapedData, 0x7d, 0x01)
		case 0x7e:
			// Turn 0x7e into 0x7d02
			escapedData = append(escapedData, 0x7d, 0x02)
		default:
			escapedData = append(escapedData, data[i])
		}
	}
	return escapedData
}

// XOR all bytes in byte array together
func xorbytes(input []byte) byte {
	var result byte
	for _, b := range input {
		result ^= b
	}
	return result
}

// Get current UTC+8 time and return in BCD format
func GetCNTimeAsBCD() []byte {
	t := time.Now().UTC()
	tz := time.FixedZone("UTC+8", 8*60*60)
	ct := t.In(tz)
	return []byte{
		toBCD(ct.Year() % 100),
		toBCD(int(ct.Month())),
		toBCD(ct.Day()),
		toBCD(ct.Hour()),
		toBCD(ct.Minute()),
		toBCD(ct.Second()),
	}
}

// Convert int to BCD
// Example 12 -> 0x12
func toBCD(n int) byte {
	return byte(((n / 10) << 4) | (n % 10))
}
