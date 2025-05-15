package gt06

import (
	"banjo.dev/trackerr/internal/model"
	"banjo.dev/trackerr/internal/utils"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"
)

// Constants specific to GT06
const (
	StartByte           byte          = 0x78
	StartByteExtended   byte          = 0x79
	EndByte1            byte          = 0x0D
	EndByte2            byte          = 0x0A
	coordinatePrecision float64       = 30000 * 60
	MsgTypeLogin        uint8         = 0x01
	MsgTypeLocation     uint8         = 0x12
	MsgTypeHeartbeat    uint8         = 0x13
	MsgTypeCmdResponse  uint8         = 0x15
	MsgTypeAlarm        uint8         = 0x16
	MsgTypeLocation4g   uint8         = 0x22
	MsgTypeCmdSend      uint8         = 0x80
	MsgTypeIMSI         uint8         = 0x90
	MsgTypeICCID        uint8         = 0x94
	HeartbeatInterval   time.Duration = 5 * time.Minute
)

var alarmTypes = map[uint8]string{
	0x00: ":Normal",
	0x01: ":SOS",
	0x02: ":Power Failure",
	0x03: ":Vibration",
	0x04: ":Entering Fence",
	0x05: ":Exiting Fence",
	0x06: ":Speeding",
	0x07: ":High Temperature",
	0x08: ":Low Temperature",
	0x09: ":Displacement",
	0x13: ":Anti-Tamper",
	0x26: ":Rapid Acceleration",
	0x27: ":Rapid Deacceleration",
	0x28: ":Sharp Turn",
	0x29: ":Collision",
	0x0E: ":Low Battery",
	0xFA: ":Door closed",
	0xFB: ":Door opened",
	0xFC: ":AC off",
	0xFD: ":AC on",
	0xFE: ":ACC ignition",
	0xFF: ":ACC flameout",
}

// Perform GT06 authentication after receiving first packet p
func PerformAuth(conn net.Conn, p model.Packet) (string, error) {
	if p.PacketType != uint16(MsgTypeLogin) {
		return "", fmt.Errorf("Invalid login message type")
	}
	imei := hex.EncodeToString(p.Payload)[1:] //Ignore first byte,
	// Send login response
	SendMsg(conn, false, MsgTypeLogin, []byte{}, p.SerialNumber)
	return imei, nil
}

// Parse GT06 message frame
func ParseMsg(conn io.Reader, extended bool) (model.Packet, error) {
	var p model.Packet
	var pl []byte
	var err error

	// Packet length is 1 bytes if start=0x7878 and 2 bytes if start=0x7979
	if extended {
		pl, err = utils.ReadBytes(conn, 2)
		if err != nil {
			return p, fmt.Errorf("Failed to read 2 byte packet length: %v", err)
		}
		p.PayloadLength = binary.BigEndian.Uint16(pl)
	} else {
		pl, err = utils.ReadBytes(conn, 1)
		if err != nil {
			return p, fmt.Errorf("Failed to read 1 byte packet length: %v", err)
		}
		p.PayloadLength = uint16(pl[0])
	}

	// Read protocol number
	pn, err := utils.ReadBytes(conn, 1)
	if err != nil {
		return p, fmt.Errorf("Failed to read protocol number: %v", err)
	}

	// Map protocol number as packet type
	p.PacketType = uint16(pn[0])

	// Read packet data
	p.Payload, err = utils.ReadBytes(conn, int(p.PayloadLength)-5)
	if err != nil {
		return p, fmt.Errorf("Error reading payload: %v", err)
	}

	// Read trailer
	trailer, err := utils.ReadBytes(conn, 4)
	if err != nil {
		return p, fmt.Errorf("Error reading trailer: %v", err)
	}
	p.SerialNumber = binary.BigEndian.Uint16(trailer[0:2])
	p.ErrorCheck = binary.BigEndian.Uint16(trailer[2:4])

	// Read stop byte
	if end, err := utils.ReadBytes(conn, 2); err != nil || end[0] != 0x0d || end[1] != 0x0a {
		return p, fmt.Errorf("Invalid stop bytes: %v", err)
	}

	// Put data for used for CRC in buffer
	buf := bytes.NewBuffer([]byte(""))
	buf.Write(pl)
	buf.Write(pn)
	buf.Write(p.Payload)
	buf.Write(trailer[0:2])

	// Calculate CRC code
	crcdata := buf.Bytes()
	if utils.CRCITU(crcdata) != p.ErrorCheck {
		return p, fmt.Errorf("Invalid error check code")
	}

	return p, nil
}

// Parse command response message
func ParseCmdRes(payload []byte) (string, uint32) {
	rlen := uint8(payload[0])
	return string(payload[5 : 5+rlen-4]), binary.BigEndian.Uint32(payload[1:5])
}

// Parse alarm message
func ParseAlarmMsg(payload []byte) (model.Locationdata, string) {
	// Parse location data section
	ld := ParseLocationMsg(payload)
	// Lookup alarm name
	if name, ok := alarmTypes[payload[31]]; ok {
		return ld, name
	}
	return ld, "Unknown"
}

func ParseLocationMsg(payload []byte) model.Locationdata {
	var ld model.Locationdata
	ld.Timestamp = parseTime(payload[0:6])
	gpsSection := payload[6:18]
	ld.Lat = binary.BigEndian.Uint32(gpsSection[1:5])
	ld.Lon = binary.BigEndian.Uint32(gpsSection[5:9])
	ld.Speed = uint16(gpsSection[9])
	front := gpsSection[10] & 3
	ld.Heading = binary.BigEndian.Uint16([]byte{front, gpsSection[11]})
	utils.StdLatLon(&ld, coordinatePrecision)
	return ld
}

// Convert time from [yy,mm,dd,hh,mm,ss] to unix time
func parseTime(timeBytes []byte) int64 {
	return int64(time.Date(
		2000+int(timeBytes[0]),
		time.Month(timeBytes[1]),
		int(timeBytes[2]),
		int(timeBytes[3]),
		int(timeBytes[4]),
		int(timeBytes[5]),
		0, time.UTC).Unix())
}

// Send message in GT06 format
func SendMsg(conn net.Conn, extended bool, msgtype uint8, payload []byte, serialnum uint16) {
	buf := bytes.NewBuffer([]byte{})
	// Write message length
	mlen := len(payload) + 5
	// Set ecoffset which indicates how many of the first bytes should be excluded from CRCITU
	var ecoffset int
	if extended {
		buf.Write([]byte{StartByteExtended, StartByteExtended})
		// Extended messages use 2 bytes for message length
		binary.Write(buf, binary.BigEndian, mlen)
		ecoffset = 3
	} else {
		buf.Write([]byte{StartByte, StartByte})
		// Non-extended messages use 1 byte for message length
		buf.WriteByte(byte(mlen))
		ecoffset = 2
	}
	buf.WriteByte(byte(msgtype))
	buf.Write(payload)
	binary.Write(buf, binary.BigEndian, serialnum)
	binary.Write(buf, binary.BigEndian, utils.CRCITU(buf.Bytes()[ecoffset:]))
	buf.Write([]byte{EndByte1, EndByte2})
	conn.Write(buf.Bytes())
	return
}

// Send command message
func SendCmd(conn net.Conn, content string, serialNumber uint16, cmdid uint32) {
	cmdbuf := bytes.NewBuffer([]byte{})
	cmdbuf.WriteByte(byte(len(content) + 4))
	// Bytes to map corresponding response. Set to static since not used in response parser
	binary.Write(cmdbuf, binary.BigEndian, cmdid)
	cmdbuf.Write([]byte(content))
	SendMsg(conn, false, MsgTypeCmdSend, cmdbuf.Bytes(), serialNumber)
}
