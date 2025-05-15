package protocols

import (
	"banjo.dev/trackerr/internal/model"
	"banjo.dev/trackerr/internal/protocols/gt06"
	"banjo.dev/trackerr/internal/protocols/jt808"
	"banjo.dev/trackerr/internal/utils"
	"fmt"
	"log"
	"net"
	"time"
)

// Detect protocol and authenticate accordingly
func PerformAuth(conn net.Conn) (string, error, int) {
	p, err, protocol := ParseMsg(conn, 60*time.Second)
	if err != nil {
		return "", fmt.Errorf("Failed to parse: %v", err), 0
	}
	switch protocol {
	case utils.ProtocolTypeJT808:
		id, err := jt808.PerformAuth(conn, p)
		return id, err, protocol
	case utils.ProtocolTypeGT06:
		id, err := gt06.PerformAuth(conn, p)
		return id, err, protocol
	}
	return "", fmt.Errorf("Unknown protocol"), 0
}

// Read start byte/bytes and pass to corresponding parser
func ParseMsg(conn net.Conn, maxWait time.Duration) (model.Packet, error, int) {
	// Set deadline to make more and less blocking
	// The deadline is high when waiting for login and low for other cases to allow goroutine to read command channel
	conn.SetReadDeadline(time.Now().Add(maxWait))
	start := make([]byte, 1)
	_, err := conn.Read(start)
	if err != nil {
		return model.Packet{}, err, 0
	}

	switch start[0] {
	case jt808.StartByte: // Start byte used for JT808
		p, err := jt808.ParseMsg(conn, utils.NullTime{IsSet: false})
		return p, err, utils.ProtocolTypeJT808
	case gt06.StartByte, gt06.StartByteExtended: //Start bytes used for GT06

		// Verify that also the second byte is valid
		start2 := make([]byte, 1)
		conn.SetReadDeadline(time.Now().Add(maxWait * time.Second))
		_, err := conn.Read(start2)
		if err != nil {
			return model.Packet{}, err, 0
		}
		if start[0] != start2[0] {
			log.Printf("Invalid secondary byte: %v %v\n", start[0], start2[0])
			return model.Packet{}, fmt.Errorf("Invalid second start byte"), utils.ProtocolTypeJT808
		}

		p, err := gt06.ParseMsg(conn, start[0] == gt06.StartByteExtended)
		return p, err, utils.ProtocolTypeGT06
	default:
		return model.Packet{}, fmt.Errorf("Invalid start bytes: %v byte was:%v", err, start[0]), 0
	}
}
