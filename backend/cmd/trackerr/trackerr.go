package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"

	"banjo.dev/trackerr/internal/api"
	"banjo.dev/trackerr/internal/database"
	"banjo.dev/trackerr/internal/model"
	"banjo.dev/trackerr/internal/protocols"
	"banjo.dev/trackerr/internal/protocols/gt06"
	"banjo.dev/trackerr/internal/protocols/jt808"
	"banjo.dev/trackerr/internal/utils"
	"github.com/joho/godotenv"
)

// @title           Trackerr
// @version         1.0
// @description     API for Trackerr service.
// @Schemes         http https
// @host            api.banjo.dev:8080
// @BasePath        /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	// Load .env config values
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	SERVER_IP := os.Getenv("SERVER_IP")
	TRACKERCOM_PORT := os.Getenv("TRACKERCOM_PORT")
	API_PORT := os.Getenv("API_PORT")
	API_CERT := os.Getenv("API_CERT")
	API_CERTKEY := os.Getenv("API_CERTKEY")

	// Include time when using log.print
	log.SetFlags(log.LstdFlags)

	log.Println("Main: Connecting to database")
	database.ConnectToDB()
	defer database.CloseDB()
	// Create map of substitutions to be used for models, to replace <ip>
	// with actual ip and <port> with actual port
	submap := map[string]string{"<ip>": SERVER_IP, "<port>": TRACKERCOM_PORT}
	database.SetCommandSubstituation(submap)
	// Create trackerManager
	trackerManager := &model.TrackerManager{
		Handlers:     make(map[string]*model.TrackerHandler),
		EventHandler: make(chan model.Locationdata, 100),
		CommandQueue: make(chan model.TrackerCommand, 100),
	}
	go api.StartAPI(trackerManager, API_PORT, API_CERT, API_CERTKEY)
	go eventHandler(trackerManager.EventHandler)
	go commandHandler(trackerManager)
	tcpserver(TRACKERCOM_PORT, trackerManager)
}

// Handles commands, by routing commands from the API to the specified tracker handler
func commandHandler(tm *model.TrackerManager) {
	for {
		cmd := <-tm.CommandQueue
		handler, ok := tm.Handlers[cmd.TrackerId]
		if !ok {
			cmd.Response <- "Failed to run command, since tracker is not connected"
			continue
		}
		handler.CommandQueue <- cmd
	}
}

// Handles location events by reading from the event channel and then storing them in DB
func eventHandler(events chan model.Locationdata) {
	for {
		tle := <-events
		log.Printf("Event: %v\n", tle)
		if tle.Lat == 0 && tle.Lon == 0 {
			log.Printf("Received location event with empty coordinates\n")
			return

		}
		if err := database.InsertLocationRecord(tle); err != nil {
			log.Printf("EventHandler: Error: %v\n", err)

		}
	}
}

// Listens to TRACKERCOM_PORT, and when a tracker connects passes it to a new tracker handler
func tcpserver(TRACKERCOM_PORT string, tm *model.TrackerManager) {
	// Listen on TRACKERCOM_PORT
	l, err := net.Listen("tcp", ":"+TRACKERCOM_PORT)
	if err != nil {
		log.Println("TCPServer: Error listening: ", err.Error())
		log.Fatal(err)
	}
	defer l.Close()
	log.Println("TCPServer: Listening on port: " + TRACKERCOM_PORT)

	for {
		// Wait for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// Cast conn to *net.TCPConn
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			// Disable keepalive
			tcpConn.SetKeepAlive(false)
		}

		// Pass connection to new tracker handler
		go handleTracker(tm, conn)
	}
}

func handleTracker(tm *model.TrackerManager, conn net.Conn) {
	// Authenticate tracker
	trackerId, protocol, err := protocols.PerformAuth(conn)
	if err != nil {
		log.Println("Handshake failed: ", err)
		conn.Close()
		return
	}
	// Close connection if tracker is not registered or not enabled
	if !database.IsTrackerEnabled(trackerId) {
		log.Printf("%v: Tracker is not registered or disabled\n", trackerId)
		conn.Close()
		return
	}
	// Create trackerHandler
	handler := &model.TrackerHandler{
		Id:           trackerId,
		Conn:         conn,
		CommandQueue: make(chan model.TrackerCommand, 10),
		EventHandler: tm.EventHandler,
		SerialNumber: 1,
		DoneFlag:     make(chan bool),
	}

	// Store trackerHandler in trackerManager
	tm.Mu.Lock()
	val, ok := tm.Handlers[trackerId]
	// If connection with same trackerId exists, signal DoneFlag
	if ok {
		val.DoneFlag <- true
	}
	tm.Handlers[trackerId] = handler
	tm.Mu.Unlock()
	database.UpdateLastConnected(trackerId, time.Now().UTC().Unix())
	// Pass connection to protocol specific handler
	switch protocol {
	case utils.ProtocolTypeJT808:
		handleJT808Connection(handler)
	case utils.ProtocolTypeGT06:
		handleGT06Connection(handler)
	}
	// Remove from handler if still in trackerManager.
	// If it's killed by the done flag, is likely overwritten in tm.Handlers and therefore should NOT  be removed.
	tm.Mu.Lock()
	if handler == tm.Handlers[trackerId] {
		delete(tm.Handlers, trackerId)
	}
	tm.Mu.Unlock()
	log.Printf("Remvoved handler for %v\n", trackerId)
}

func handleGT06Connection(t *model.TrackerHandler) {
	defer log.Printf("%v: Connection has been closed\n", t.Id)
	defer t.Conn.Close()

	log.Printf("%v: Device has conencted!\n", t.Id)
	resChannelMap := make(map[uint32]chan string)
	heartbeatTimer := time.NewTimer(gt06.HeartbeatInterval + time.Minute)
	defer heartbeatTimer.Stop()

	for {
		select {
		// If done flag set, kill connection
		case <-t.DoneFlag:
			log.Printf("%v: Closing connection since Done flag is set\n", t.Id)
			return
		// Close connection if heartbeat not received in timely manner
		case <-heartbeatTimer.C:
			log.Printf("%v: Closing connection since heartbeat was not received\n", t.Id)
			return
		// If command in queue, send it
		case cmd := <-t.CommandQueue:
			gt06.SendCmd(t.Conn, cmd.Payload, t.SerialNumber, uint32(t.SerialNumber))
			resChannelMap[uint32(t.SerialNumber)] = cmd.Response
			t.SerialNumber++
			log.Printf("%v: Sent: %v\n", t.Id, cmd)
		default:
			// Read and parse packet, using a 1s deadline
			p, _, err := protocols.ParseMsg(t.Conn, 1*time.Second)

			if err != nil {
				// Continue if there is no packet to read
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}
				// Stop handler if tracker want to end connection
				if err == io.EOF {
					return
				}
				log.Printf("%v: Failed to parse packet:%v\n", err, t.Id)
				continue
			}
			switch uint8(p.PacketType) {
			// Location update
			case gt06.MsgTypeLocation, gt06.MsgTypeLocation4g:
				ld := gt06.ParseLocationMsg(p.Payload)
				ld.TrackerId = t.Id
				ld.Timestamp = time.Now().Unix()
				log.Printf("%v: Position: %v\n", t.Id, utils.StringifyCoordinates(ld.Lat, ld.Lon))
				t.EventHandler <- ld
			// Heartbeat
			case gt06.MsgTypeHeartbeat:
				log.Printf("%v: Received heartbeat\n", t.Id)
				// Reset heartbeat timer
				heartbeatTimer.Reset(gt06.HeartbeatInterval + time.Minute)
				// Potential to implement parser for terminal info, voltage level, gsm signal strength, external voltage and language
				gt06.SendMsg(t.Conn, false, gt06.MsgTypeHeartbeat, []byte{}, p.SerialNumber)
			// Server cmd response
			case gt06.MsgTypeCmdResponse:
				r, id := gt06.ParseCmdRes(p.Payload)
				log.Printf("%v: Server Response:%v\n", t.Id, r)

				// Find corresponding response channel in map
				rChannel, ok := resChannelMap[uint32(id)]
				// Check if it was found
				if !ok {
					log.Printf("%v: Received response but response queue was empty\n", t.Id)
					continue
				}
				// Send the reponse to it, if it was found
				rChannel <- r
				// Remove from map when it has sent the response
				delete(resChannelMap, uint32(id))
			// Alarm
			case gt06.MsgTypeAlarm:
				ld, alarm := gt06.ParseAlarmMsg(p.Payload)
				log.Printf("%v: Received %v alarm\n", t.Id, alarm)
				log.Printf("%v: Position: %v\n", t.Id, utils.StringifyCoordinates(ld.Lat, ld.Lon))
				ld.TrackerId = t.Id
				t.EventHandler <- ld
			// IMSI
			case gt06.MsgTypeIMSI:
				log.Printf("%v: Terminal sending IMSI number\n", t.Id)
			// ICCID
			case gt06.MsgTypeICCID:
				log.Printf("%v: Terminal sending ICCID number\n", t.Id)
			// Unknown
			default:
				log.Printf("%v: Unknown protocol number: %x\nPayload:%v", t.Id, p.PacketType, p.Payload)
			}
		}
	}
}

func handleJT808Connection(t *model.TrackerHandler) {
	defer log.Printf("%v: Connection has been closed\n", t.Id)
	defer t.Conn.Close()
	log.Printf("%v: Device has conencted!\n", t.Id)

	resChannelQueue := make([]chan string, 0)
	heartbeatTimer := time.NewTimer(jt808.HeartbeatInterval + time.Minute)
	defer heartbeatTimer.Stop()

	for {
		select {
		// If done flag set, kill connection
		case <-t.DoneFlag:
			log.Printf("%v: Closing connection since Done flag is set\n", t.Id)
			return
		// Close connection if heartbeat not received in timely manner
		case <-heartbeatTimer.C:
			log.Printf("%v: Closing connection since heartbeat was not received\n", t.Id)
			return

		// If command in queue, send it
		case cmd := <-t.CommandQueue:
			jt808.SendCmd(t.Conn, cmd.Payload, t.Id, t.SerialNumber)
			t.SerialNumber++
			log.Printf("%v: Sent: %v\n", t.Id, cmd)
			// Add response channel to queue
			resChannelQueue = append(resChannelQueue, cmd.Response)

		default:
			p, _, err := protocols.ParseMsg(t.Conn, 1*time.Second)
			if err != nil {
				// No TCP packet in buffer
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}
				// Client wants to terminate the connection
				if err == io.EOF {
					return
				}
				log.Println("Failed to parse packet:", err)
				continue
			}
			switch p.PacketType {

			case jt808.MsgTypeTermUniversalRes: // Universal terinal response - ignore
				continue
			case jt808.MsgTypeHeartbeat: // Heartbeat
				log.Println("Recevied heartbeat")
				// Reset heartbeat timer
				heartbeatTimer.Reset(jt808.HeartbeatInterval + time.Minute)
				jt808.SendUniversalRes(t.Conn, p.PacketType, p.SerialNumber, jt808.ResultSuccess, t.Id)
			case jt808.MsgTypeLogout: // log out
				if err := database.RemoveAuthCode(t.Id); err != nil {
					log.Println(err)
				}
				return
			case jt808.MsgTypeLocation: // Position info report
				log.Println("Recevied position info")
				jt808.SendUniversalRes(t.Conn, p.PacketType, p.SerialNumber, jt808.ResultSuccess, t.Id)
				ld := jt808.ParseLocationMsg(p.Payload)
				utils.StdLatLon(&ld, jt808.CoordinatePrecision)
				log.Printf("%v: Position: %v\n", t.Id, utils.StringifyCoordinates(ld.Lat, ld.Lon))
				ld.TrackerId = t.Id
				ld.Timestamp = time.Now().Unix()
				t.EventHandler <- ld
			case jt808.MsgTypeVersionInfo: // Version info packet
				log.Println("Recevied version info")
				payload := append(jt808.GetCNTimeAsBCD(), []byte{0, 0, 0, 0, 0}...)
				log.Println("Chinese time:", payload)
				jt808.SendMsg(t.Conn, jt808.MsgTypeVersionInfoRes, payload, p.SerialNumber, t.Id)
			case jt808.MsgTypeLocationBatch: // Position info batch report --NOT IMPLEMENTED
				jt808.SendUniversalRes(t.Conn, p.PacketType, p.SerialNumber, jt808.ResultSuccess, t.Id)
			case jt808.MsgTypeUpstreamData: // Upstream data --NOT IMPLEMENTED
				jt808.SendUniversalRes(t.Conn, p.PacketType, p.SerialNumber, jt808.ResultSuccess, t.Id)
			case jt808.MsgTypeCmdRes: // Command Response
				r := jt808.ParseCmdRes(p.Payload)
				log.Printf("%v: Received command response: %v\n", t.Id, r)
				// Check if response channel queue is empty
				if len(resChannelQueue) == 0 {
					log.Printf("%v: Received response but response queue was empty\n", t.Id)
					continue
				}
				// Send response to first response channel in queue
				resChannelQueue[0] <- r
				// Pop the head
				resChannelQueue = resChannelQueue[1:]
			default:
				log.Printf("%v: Unknown protocol number: %x\nPayload:%v", t.Id, p.PacketType, p.Payload)
			}
		}
	}
}
