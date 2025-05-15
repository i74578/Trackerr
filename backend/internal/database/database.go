package database

import (
	"banjo.dev/trackerr/internal/model"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
)

const file string = "database.db"

var db *sql.DB
var submap map[string]string

type Repository interface {
	VerifyAPIKey(key string) bool
	VerifyTracker(key string) bool
	GetTrackerLocationHistory(trackerID string) []model.Locationdata
}

func GetTrackerLocationHistory(trackerID string) ([]model.Locationdata, error) {
	var ld []model.Locationdata
	// Create and run SQL query
	rows, err := db.Query("SELECT * FROM location_data WHERE trackerId = ?", trackerID)
	if err != nil {
		return ld, fmt.Errorf("No history found for tracker:%v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var i model.Locationdata
		if err := rows.Scan(&i.EntryId, &i.TrackerId, &i.Timestamp, &i.Lat, &i.Lon, &i.Speed, &i.Heading); err != nil {
			log.Fatal(err)
		}
		log.Printf("ID:%v, Tracker: %v, Lat:%v, Lon:%v\n", i.EntryId, i.TrackerId, i.Lat, i.Lon)
		ld = append(ld, i)
	}
	return ld, nil
}

func GetLocation(TrackerID string) (model.Locationdata, error) {
	var ld model.Locationdata
	// Create and run SQL query
	row := db.QueryRow("SELECT timestamp,lat,lon,speed,heading FROM location_data WHERE trackerId = ? ORDER BY timestamp DESC LIMIT 1", TrackerID)
	if err := row.Scan(&ld.Timestamp, &ld.Lat, &ld.Lon, &ld.Speed, &ld.Heading); err != nil {
		if err == sql.ErrNoRows {
			return ld, fmt.Errorf("No location entry found for tracker: %v", err)
		}
		log.Fatal(err)
	}
	return ld, nil
}

func InsertLocationRecord(ld model.Locationdata) error {
	// Create and run SQL query
	_, err := db.Exec("INSERT INTO location_data (trackerId,timestamp,lat,lon,speed,heading) VALUES (?,?,?,?,?,?)", ld.TrackerId, ld.Timestamp, ld.Lat, ld.Lon, ld.Speed, ld.Heading)
	if err != nil {
		return fmt.Errorf("Failed to insert location record: %v", err)
	}
	return nil
}

// Users
func GetUserByAPIKey(apikey string) (model.User, error) {
	var user model.User
	// Create and run SQL query
	row := db.QueryRow("SELECT id,name,apikey,admin,enabled FROM users WHERE apikey = ?", apikey)
	if err := row.Scan(&user.Id, &user.Name, &user.Apikey, &user.Admin, &user.Enabled); err != nil {
		if err == sql.ErrNoRows {
			return user, fmt.Errorf("User not found: %v", err)
		}
		log.Fatal(err)
	}
	return user, nil
}

// Trackers
func GetTrackers() []model.TrackerWithLocation {
	return GetTrackersByFilter("", nil)
}

func GetTrackersByUserId(userId int) []model.TrackerWithLocation {
	return GetTrackersByFilter(" WHERE t.owner = ?", []interface{}{userId})
}

func GetTrackersByUserAndTrackerId(userId int, trackerId string) []model.TrackerWithLocation {
	return GetTrackersByFilter(" WHERE t.owner = ? AND t.id = ?", []interface{}{userId, trackerId})
}

func GetTrackerByName(name string) (model.TrackerWithLocation, error) {
	trackers := GetTrackersByFilter(" WHERE t.name = ?", []interface{}{name})
	if len(trackers) == 0 {
		return model.TrackerWithLocation{}, fmt.Errorf("Requested tracker was not found")
	}
	return trackers[0], nil
}

func GetTracker(TrackerID string) (model.TrackerWithLocation, error) {
	trackerSplice := GetTrackersByFilter(" WHERE t.id = ?", []interface{}{TrackerID})
	if len(trackerSplice) != 1 {
		return model.TrackerWithLocation{}, fmt.Errorf("Requested tracker was not found")

	}
	return trackerSplice[0], nil
}

func GetTrackersByFilter(whereClause string, args []interface{}) []model.TrackerWithLocation {
	var t []model.TrackerWithLocation
	// Create and run SQL query. Query joins each tracker with latest associated location data
	rows, err := db.Query("WITH latest_ld AS ( SELECT trackerId, timestamp, lat, lon, speed, heading, ROW_NUMBER() OVER ( PARTITION BY trackerId ORDER BY timestamp DESC ) AS rn FROM location_data ) SELECT t.id, t.name, t.owner, t.phoneNumber, t.model, t.enabled, t.lastConnected, ld.timestamp, ld.lat, ld.lon, ld.speed, ld.heading FROM trackers AS t LEFT JOIN latest_ld AS ld ON ld.trackerId = t.id AND ld.rn = 1"+whereClause, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var twl model.TrackerWithLocation
		var timestamp *int64
		var lat *uint32
		var lon *uint32
		var speed *uint16
		var heading *uint16
		// Scan tracker data into twl and location data into seperate variables
		if err := rows.Scan(&twl.Tracker.Id, &twl.Name, &twl.Owner, &twl.PhoneNumber, &twl.Model, &twl.Enabled, &twl.LastConnected, &timestamp, &lat, &lon, &speed, &heading); err != nil {
			log.Fatal(err)
		}
		// If tracker has location data, then create and append location data to twl
		if timestamp != nil {
			var locationdata model.Locationdata
			locationdata.Timestamp = *timestamp
			locationdata.Lat = *lat
			locationdata.Lon = *lon
			locationdata.Speed = *speed
			locationdata.Heading = *heading
			twl.Ld = &locationdata
		} else {
			twl.Ld = nil
		}
		t = append(t, twl)
	}
	return t
}

func IsTrackerEnabled(trackerId string) bool {
	var tid string
	// Create and run SQL query
	row := db.QueryRow("SELECT id from trackers WHERE id = ? AND enabled = 1", trackerId)
	if err := row.Scan(&tid); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		return false

	}
	return true
}

func RegisterTracker(t model.Tracker) model.TrackerRegistrationResult {
	// Create and run SQL query
	_, err := db.Exec("INSERT INTO trackers (id,name,owner,phoneNumber,model,enabled) VALUES (?,?,?,?,?,?)", t.Id, t.Name, t.Owner, t.PhoneNumber, t.Model, t.Enabled)
	if err != nil {
		log.Printf("Failed to insert tracker into database: %v\n", err)
		return handleTrackerRegistrationError(t, err)

	}
	return model.TrackerRegistrationSuccess
}

func handleTrackerRegistrationError(t model.Tracker, err error) model.TrackerRegistrationResult {
	trackerWithSameId, errId := GetTracker(t.Id)
	trackerWithSameName, errName := GetTrackerByName(t.Name)

	// Unknown error, if insert failed but tracker with same id/name not found
	if errId != nil && errName != nil {
		return model.TrackerRegistrationUnknownError
	}
	if t.Id == trackerWithSameId.Id && t.Name == trackerWithSameId.Name {
		return model.TrackerRegistrationIdenticalExists
	}
	if errId == nil && t.Owner != trackerWithSameId.Owner {
		return model.TrackerRegistrationIdUsedByOtherUser
	}
	if errName == nil && t.Owner != trackerWithSameName.Owner {
		return model.TrackerRegistrationNameUsedByOtherUser
	}
	if errId != nil {
		return model.TrackerRegistrationIdUsedByOwner
	}
	if errName != nil {
		return model.TrackerRegistrationNameUsedByOwner
	}
	return model.TrackerRegistrationUnknownError
}

func DeregisterTracker(TrackerID string) error {
	// Create and run SQL query
	res, err := db.Exec("DELETE FROM trackers WHERE id = ?", TrackerID)
	if err != nil {
		return fmt.Errorf("Failed to remote tracker from database: %v", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected != 1 {
		return fmt.Errorf("Rows affected is not equal to 1: %v", err)
	}

	return nil
}

func SetTrackerEnabled(TrackerID string, enabledBool bool) error {
	enabled := 0
	if enabledBool {
		enabled = 1
	}
	// Create and run SQL query
	_, err := db.Exec("UPDATE trackers SET enabled = ? WHERE id = ?", enabled, TrackerID)
	if err != nil {
		return fmt.Errorf("Failed to update enabled state of %v: %v", TrackerID, err)
	}
	return nil

}

func UpdateLastConnected(TrackerID string, timestamp int64) error {
	// Create and run SQL query
	_, err := db.Exec("UPDATE trackers SET lastConnected = ? WHERE id = ?", timestamp, TrackerID)
	if err != nil {
		return fmt.Errorf("Failed to update last connected property of %v: %v", TrackerID, err)
	}
	return nil
}

// Tracker Models
func GetModelsByFilter(whereClause string, args []interface{}) []model.Model {
	var m []model.Model
	// Create and run SQL query
	rows, err := db.Query("SELECT name,init_commands,success_keywords FROM models"+whereClause, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var i model.Model
		if err := rows.Scan(&i.Name, &i.Init_commands, &i.Success_keywords); err != nil {
			log.Fatal(err)
		}
		// Substitute variables in commands and keywords
		i.Init_commands = SubstituteCommand(i.Init_commands)
		i.Success_keywords = SubstituteCommand(i.Success_keywords)

		m = append(m, i)
	}
	return m
}

func CreateModel(m model.Model) error {
	// Create and run SQL query
	_, err := db.Exec("INSERT INTO models (name,init_commands,success_keywords) VALUES (?,?,?)", m.Name, m.Init_commands, m.Success_keywords)
	if err != nil {
		return fmt.Errorf("Failed to create model: %v", err)
	}
	return nil
}

func GetModels() []model.Model {
	return GetModelsByFilter("", nil)

}

func GetModel(name string) (model.Model, error) {
	m := GetModelsByFilter(" WHERE name = ?", []interface{}{name})
	if len(m) != 1 {
		return model.Model{}, fmt.Errorf("getModel found more or less than 1 model")
	}
	return m[0], nil
}

func DeleteModel(name string) error {
	// Create and run SQL query
	_, err := db.Exec("DELETE FROM models WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("Failed to remote tracker from database: %v", err)
	}
	return nil
}

// JT808 Authcodes
func FetchAuthCode(trackerId string) (model.AuthCode, error) {
	var ac model.AuthCode
	// Create and run SQL query
	row := db.QueryRow("SELECT trackerId,code from jt808_authcodes WHERE trackerId = ?", trackerId)
	if err := row.Scan(&ac.TrackerId, &ac.Code); err != nil {
		if err == sql.ErrNoRows {
			return ac, fmt.Errorf("No auth code found for tracker: %v", err)
		}
		log.Fatal(err)
	}
	return ac, nil
}

func SaveAuthCode(t model.AuthCode) error {
	// Create and run SQL query
	_, err := db.Exec("INSERT OR REPLACE INTO jt808_authcodes (trackerId,code) VALUES (?,?)", t.TrackerId, t.Code)
	if err != nil {
		return fmt.Errorf("Failed to insert auth code into database: %v", err)
	}
	return nil
}

func RemoveAuthCode(trackerId string) error {
	// Create and run SQL query
	_, err := db.Exec("DELETE FROM jt808_authcodes WHERE trackerId = ?", trackerId)
	if err != nil {
		return fmt.Errorf("Failed to remove authcode from database: %v", err)
	}
	return nil
}

// Database
func ConnectToDB() {
	var err error
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", file)
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatalf("couldn’t enable WAL mode: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err != nil {
		log.Fatal(err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(err)
	}
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}

func SetCommandSubstituation(submapin map[string]string) {
	submap = submapin
}

// Substitude text in command based on submap
// This is used to fill in variables in provisioning SMS messages
func SubstituteCommand(command string) string {
	for from, to := range submap {
		command = strings.ReplaceAll(command, from, to)
	}
	return command
}
