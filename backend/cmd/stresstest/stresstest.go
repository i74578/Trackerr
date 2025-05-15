package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Struct for each tracker simulator
type trackerSim struct {
	id             string
	finished       chan bool
	uploadInterval time.Duration
	lifetime       int
}

var (
	ipAliasList = []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("172.232.141.217"),
		net.ParseIP("192.168.159.162"),
	}
)

func main() {
	// Interval of how often a tracker should send a location update
	uploadInterval := 30 * time.Second
	// Total length of simulation
	simulationLength := 180 * time.Second
	// Count of simulated GT06 GPS trackers
	simulationCount := 10000

	simulators := make([]trackerSim, simulationCount)

	// Create simulators splice and fill in trackers
	for i := range simulators {
		id := fmt.Sprintf("%013d", i)
		//fmt.Printf("%v\n",id)
		simulators[i] = trackerSim{id: id, finished: make(chan bool), uploadInterval: uploadInterval, lifetime: int(simulationLength / uploadInterval)}
		simulators[i].id = id
		// Register tracker using API
		if !registerTracker(id) {
			log.Fatal("Abort: Failed to register tracker")
		}
	}

	startTime := time.Now()
	// Start all trackers as goroutines, and distribute the start time equally in a uploadInterval. This should result in a uniform distribution of sent update location packets based on time
	for i := range simulators {
		go simulateTracker(simulators[i])
		time.Sleep(uploadInterval / time.Duration(simulationCount))
	}

	// Wait for all simulators to be finished
	for i := range simulators {
		<-simulators[i].finished
	}
	latency := time.Since(startTime)
	fmt.Printf("Test took:%v\n", latency)

	fmt.Println("Press the Enter Key to start the deletion process of the registerd trackers")
	fmt.Scanln() // wait for Enter Key

	// Deregister all trackers
	for i := range simulators {
		id := simulators[i].id
		if !deregisterTracker(id) {
			log.Fatal("Abort: Failed to deregister tracker")
		}
	}
}

func simulateTracker(sim trackerSim) {
	// Convery ID from string to int
	i, err := strconv.Atoi(sim.id)
	if err != nil {
		log.Fatal(err)
	}
	// Use all IP addresses of machine to prevent exhausting all ephenumeral ports
	dialer := net.Dialer{
		LocalAddr: &net.TCPAddr{IP: ipAliasList[i%len(ipAliasList)], Port: 0},
	}
	// Connect to backend
	conn, err := dialer.Dial("tcp", "127.0.0.1:5023")
	if err != nil {
		log.Fatal("Tracker failed to connect to backend")
	}
	defer conn.Close()
	// Send login packet
	conn.SetDeadline(time.Now().Add(5000 * time.Second))
	idAsBCDHex, err := hex.DecodeString("0" + sim.id)
	sendGT06Response(conn, 0x01, idAsBCDHex, 1)
	// Read response and verify the it is correct
	resbuf := make([]byte, 10)
	conn.Read(resbuf)
	sequence := []byte{0x78, 0x78, 0x05, 0x01, 0x00, 0x01, 0xd9, 0xdc, 0x0d, 0x0a}
	if !bytes.Contains(resbuf, sequence) {
		log.Fatal("Received invalid response from server")
	}
	// Send location update packet every uploadInterval for lifetime cycles
	for cycles := 0; cycles < sim.lifetime; cycles++ {
		conn.Write([]byte{0x78, 0x78, 0x2f, 0x22, 0x19, 0x4, 0x19, 0x12, 0xe, 0x32, 0xcf, 0x5, 0xf8, 0x5b, 0x28, 0x1, 0x5a, 0xdf, 0x1c, 0x0, 0x55, 0x3a, 0x0, 0xee, 0x2, 0x5d, 0xfd, 0x2, 0x76, 0x86, 0x17, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x54, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xff, 0xff, 0x0, 0xe0, 0x29, 0xe3, 0xd, 0xa})
		time.Sleep(sim.uploadInterval)
	}
	conn.Close()
	// Notify main thread that it has finished
	sim.finished <- true
}

func registerTracker(id string) bool {
	// Set HTTP Post payload
	payload := map[string]interface{}{
		"id":          id,
		"name":        id,
		"phoneNumber": "12345678",
		"model":       "W18L",
		"enabled":     true,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Failed to register: %v\n", id)
		return false
	}
	// Create POST request
	req, err := http.NewRequest("POST", "https://api.banjo.dev:8080/api/v1/trackers", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to register: %v\n", id)
		return false
	}
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "AAAAAA")
	// Create client and send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to register: %v\n", id)
		return false
	}
	// Read response body
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to register: %v\n", id)
		return false
	}
	// Fail of response code is not 201
	if resp.StatusCode != 201 {
		fmt.Printf("Failed to register: %v\n", id)
		return false
	}
	return true

}

func deregisterTracker(id string) bool {
	// Create DELETE request
	req, err := http.NewRequest("DELETE", "https://api.banjo.dev:8080/api/v1/trackers/"+id, nil)
	if err != nil {
		fmt.Printf("Failed to deregister: %v\n", id)
		return false
	}
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "AAAAAA")
	// Create client and send
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to deregister: %v\n", id)
		return false

	}
	// Read response
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to deregister: %v\n", id)
		return false
	}
	// Verify that status code is 200
	if resp.StatusCode != 200 {
		fmt.Printf("Failed to deregister: %v\n", id)
		return false
	}
	return true
}

// Function to send GT06 responses. Same function as used in Trackerr
func sendGT06Response(conn net.Conn, msgtype uint8, payload []byte, serialnum uint16) {
	buf := bytes.NewBuffer([]byte{})
	// Calculate payload length
	plen := len(payload) + 5
	// Write start bytes
	buf.Write([]byte{0x78, 0x78})
	// Write length
	buf.WriteByte(byte(plen))
	// Write message type
	buf.WriteByte(byte(msgtype))
	// Write payload
	buf.Write(payload)
	// Write serial number
	binary.Write(buf, binary.BigEndian, serialnum)
	// Write error check code
	binary.Write(buf, binary.BigEndian, crcitu(buf.Bytes()[2:]))
	// Write end bytes
	buf.Write([]byte{0x0d, 0x0a})
	// Send buffer bytes to conn
	conn.Write(buf.Bytes())
	return
}

func crcitu(data []byte) uint16 {
	var crctab16 = [256]uint16{
		0x0000, 0x1189, 0x2312, 0x329B, 0x4624, 0x57AD, 0x6536, 0x74BF,
		0x8C48, 0x9DC1, 0xAF5A, 0xBED3, 0xCA6C, 0xDBE5, 0xE97E, 0xF8F7,
		0x1081, 0x0108, 0x3393, 0x221A, 0x56A5, 0x472C, 0x75B7, 0x643E,
		0x9CC9, 0x8D40, 0xBFDB, 0xAE52, 0xDAED, 0xCB64, 0xF9FF, 0xE876,
		0x2102, 0x308B, 0x0210, 0x1399, 0x6726, 0x76AF, 0x4434, 0x55BD,
		0xAD4A, 0xBCC3, 0x8E58, 0x9FD1, 0xEB6E, 0xFAE7, 0xC87C, 0xD9F5,
		0x3183, 0x200A, 0x1291, 0x0318, 0x77A7, 0x662E, 0x54B5, 0x453C,
		0xBDCB, 0xAC42, 0x9ED9, 0x8F50, 0xFBEF, 0xEA66, 0xD8FD, 0xC974,
		0x4204, 0x538D, 0x6116, 0x709F, 0x0420, 0x15A9, 0x2732, 0x36BB,
		0xCE4C, 0xDFC5, 0xED5E, 0xFCD7, 0x8868, 0x99E1, 0xAB7A, 0xBAF3,
		0x5285, 0x430C, 0x7197, 0x601E, 0x14A1, 0x0528, 0x37B3, 0x263A,
		0xDECD, 0xCF44, 0xFDDF, 0xEC56, 0x98E9, 0x8960, 0xBBFB, 0xAA72,
		0x6306, 0x728F, 0x4014, 0x519D, 0x2522, 0x34AB, 0x0630, 0x17B9,
		0xEF4E, 0xFEC7, 0xCC5C, 0xDDD5, 0xA96A, 0xB8E3, 0x8A78, 0x9BF1,
		0x7387, 0x620E, 0x5095, 0x411C, 0x35A3, 0x242A, 0x16B1, 0x0738,
		0xFFCF, 0xEE46, 0xDCDD, 0xCD54, 0xB9EB, 0xA862, 0x9AF9, 0x8B70,
		0x8408, 0x9581, 0xA71A, 0xB693, 0xC22C, 0xD3A5, 0xE13E, 0xF0B7,
		0x0840, 0x19C9, 0x2B52, 0x3ADB, 0x4E64, 0x5FED, 0x6D76, 0x7CFF,
		0x9489, 0x8500, 0xB79B, 0xA612, 0xD2AD, 0xC324, 0xF1BF, 0xE036,
		0x18C1, 0x0948, 0x3BD3, 0x2A5A, 0x5EE5, 0x4F6C, 0x7DF7, 0x6C7E,
		0xA50A, 0xB483, 0x8618, 0x9791, 0xE32E, 0xF2A7, 0xC03C, 0xD1B5,
		0x2942, 0x38CB, 0x0A50, 0x1BD9, 0x6F66, 0x7EEF, 0x4C74, 0x5DFD,
		0xB58B, 0xA402, 0x9699, 0x8710, 0xF3AF, 0xE226, 0xD0BD, 0xC134,
		0x39C3, 0x284A, 0x1AD1, 0x0B58, 0x7FE7, 0x6E6E, 0x5CF5, 0x4D7C,
		0xC60C, 0xD785, 0xE51E, 0xF497, 0x8028, 0x91A1, 0xA33A, 0xB2B3,
		0x4A44, 0x5BCD, 0x6956, 0x78DF, 0x0C60, 0x1DE9, 0x2F72, 0x3EFB,
		0xD68D, 0xC704, 0xF59F, 0xE416, 0x90A9, 0x8120, 0xB3BB, 0xA232,
		0x5AC5, 0x4B4C, 0x79D7, 0x685E, 0x1CE1, 0x0D68, 0x3FF3, 0x2E7A,
		0xE70E, 0xF687, 0xC41C, 0xD595, 0xA12A, 0xB0A3, 0x8238, 0x93B1,
		0x6B46, 0x7ACF, 0x4854, 0x59DD, 0x2D62, 0x3CEB, 0x0E70, 0x1FF9,
		0xF78F, 0xE606, 0xD49D, 0xC514, 0xB1AB, 0xA022, 0x92B9, 0x8330,
		0x7BC7, 0x6A4E, 0x58D5, 0x495C, 0x3DE3, 0x2C6A, 0x1EF1, 0x0F78,
	}
	var fcs uint16 = 0xffff

	for i, nLength := 0, len(data); nLength > 0; i, nLength = i+1, nLength-1 {
		fcs = (fcs >> 8) ^ crctab16[(fcs^uint16(data[i]))&0xff]
	}
	return ^fcs
}
