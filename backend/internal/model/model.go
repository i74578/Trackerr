package model

import (
	"net"
	"sync"
)

type TrackerManager struct {
	Handlers     map[string]*TrackerHandler
	EventHandler chan Locationdata
	CommandQueue chan TrackerCommand
	Mu           sync.RWMutex
}

type TrackerHandler struct {
	Id           string
	CommandQueue chan TrackerCommand
	EventHandler chan Locationdata
	Conn         net.Conn
	SerialNumber uint16
	DoneFlag     chan bool
}

type TrackerCommand struct {
	TrackerId string
	Payload   string
	Response  chan string
}

// Structs for database tables
type Tracker struct {
	Id            string
	Name          string
	Owner         int
	PhoneNumber   string
	Model         string
	Enabled       bool
	LastConnected int64
}

type TrackerWithLocation struct {
	Tracker
	Ld *Locationdata
}

type User struct {
	Id      int
	Name    string
	Apikey  string
	Admin   bool
	Enabled bool
}

type Locationdata struct {
	EntryId   uint64
	TrackerId string
	Timestamp int64
	Lat       uint32
	Lon       uint32
	Speed     uint16
	Heading   uint16
}

type AuthCode struct {
	TrackerId string
	Code      string
}

type Model struct {
	Name             string
	Init_commands    string
	Success_keywords string
}

// Structs passed from between trackerr, api and database
type ServerInfo struct {
	Ip   string
	Port string
}

type TrackerRegistrationResult int

const (
	TrackerRegistrationSuccess TrackerRegistrationResult = iota
	TrackerRegistrationIdenticalExists
	TrackerRegistrationIdUsedByOwner
	TrackerRegistrationIdUsedByOtherUser
	TrackerRegistrationNameUsedByOwner
	TrackerRegistrationNameUsedByOtherUser
	TrackerRegistrationUnknownError
)

type Packet struct {
	Protocol      byte
	DeviceID      string
	PacketType    uint16
	PayloadLength uint16
	Payload       []byte
	SerialNumber  uint16
	ErrorCheck    uint16
}
