package protocols

import (
	"fmt"
	"log"
	"net"
	"time"

	"banjo.dev/trackerr/internal/model"
	"banjo.dev/trackerr/internal/protocols/gt06"
	"banjo.dev/trackerr/internal/protocols/jt808"
	"banjo.dev/trackerr/internal/utils"
)

// Detect protocol and authenticate accordingly
func PerformAuth(conn net.Conn) (string, int, error) {
	p, protocol, err := ParseMsg(conn, 60*time.Second)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse: %v", err)
	}
	switch protocol {
	case utils.ProtocolTypeJT808:
		id, err := jt808.PerformAuth(conn, p)
		return id, protocol, err
	case utils.ProtocolTypeGT06:
		id, err := gt06.PerformAuth(conn, p)
		return id, protocol, err
	}
	return "", 0, fmt.Errorf("unknown protocol")
}

// Read start byte/bytes and pass to corresponding parser
func ParseMsg(conn net.Conn, maxWait time.Duration) (model.Packet, int, error) {
	// Set deadline to make more and less blocking
	// The deadline is high when waiting for login and low for other cases to allow goroutine to read command channel
	conn.SetReadDeadline(time.Now().Add(maxWait))
	start := make([]byte, 1)
	_, err := conn.Read(start)
	if err != nil {
		return model.Packet{}, 0, err
	}

	switch start[0] {
	case jt808.StartByte: // Start byte used for JT808
		p, err := jt808.ParseMsg(conn, utils.NullTime{IsSet: false})
		return p, utils.ProtocolTypeJT808, err
	case gt06.StartByte, gt06.StartByteExtended: //Start bytes used for GT06

		// Verify that also the second byte is valid
		start2 := make([]byte, 1)
		conn.SetReadDeadline(time.Now().Add(maxWait * time.Second))
		_, err := conn.Read(start2)
		if err != nil {
			return model.Packet{}, 0, err
		}
		if start[0] != start2[0] {
			log.Printf("invalid secondary byte: %v %v\n", start[0], start2[0])
			return model.Packet{}, utils.ProtocolTypeJT808, fmt.Errorf("invalid second start byte")
		}

		p, err := gt06.ParseMsg(conn, start[0] == gt06.StartByteExtended)
		return p, utils.ProtocolTypeGT06, err
	default:
		return model.Packet{}, 0, fmt.Errorf("invalid start bytes: %v byte was:%v", err, start[0])
	}
}
