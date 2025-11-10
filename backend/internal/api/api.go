package api

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"banjo.dev/trackerr/internal/database"
	"banjo.dev/trackerr/internal/model"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Only used by swagger
type StringResultRes struct {
	Result string `json:"result"`
}
type NameRes struct {
	Name string `json:"name"`
}
type ErrorRes struct {
	Error string `json:"error"`
}

type RegisterTrackerReq struct {
	Id          string `json:"id" binding:"required,numeric,min=12,max=15"`
	Name        string `json:"name" binding:"required,min=1,max=32"`
	Owner       int    `json:"owner" binding:"numeric"`
	PhoneNumber string `json:"phoneNumber" binding:"numeric,required,min=8"`
	Model       string `json:"model" binding:"required,min=2"`
	Enabled     bool   `json:"enabled" binding:"required"`
}

type CreateModelReq struct {
	Name             string `json:"name" binding:"required,min=2"`
	Init_commands    string `json:"init_commands" binding:"required,min=1"`
	Success_keywords string `json:"success_keywords" binding:"required,min=1"`
}

type CommandReq struct {
	Command string `json:"command" binding:"required,min=1"`
}

type EnableReq struct {
	Enabled bool `json:"enabled"`
}

type LocationResponse struct {
	Timestamp *string
	Lat       *uint32
	Lon       *uint32
	Speed     *uint16
	Heading   *uint16
}

type TrackerResponse struct {
	Id            string
	Name          string
	Owner         int
	PhoneNumber   string
	Model         string
	Connected     bool
	Enabled       bool
	LastConnected string
	LocationResponse
}

var tm *model.TrackerManager

func StartAPI(tmIn *model.TrackerManager, apiPort string, certPath string, certKeyPath string) {
	tm = tmIn
	// Set gin mode from environment variable GIN_MODE (loaded via .env in main).
	// If not provided, default to release mode.
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(mode)
	}
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	api := router.Group("/api/v1")
	{
		api.Use(AuthMiddleware())
		api.GET("/whoami", whoami)

		trackers := api.Group("/trackers")
		{
			trackers.GET("", getTrackers)
			trackers.POST("", registerTracker)

			// Endpoints for specific trackers
			tracker := trackers.Group("/:id")
			{
				tracker.Use(OwnershipMiddleware())
				tracker.GET("", getTracker)
				tracker.DELETE("", deregisterTracker)
				tracker.POST("/command", sendCommand)
				tracker.GET("/location", getTrackerLocation)
				tracker.GET("/locations", getTrackerLocations)
				tracker.PUT("/enabled", setEnabled)
			}
		}

		models := api.Group("/models")
		{
			models.GET("", getModels)
			models.GET("/:name", getModel)

			protected := models.Group("", AdminOnlyMiddleware())
			{
				protected.POST("", createModel)
				protected.DELETE("", deleteModel)
			}
			models.Use(AdminOnlyMiddleware())
		}
	}
	log.Printf("API: Starting REST API, listening on port: %v\n", apiPort)
	err := router.RunTLS(":"+apiPort, certPath, certKeyPath)
	if err != nil {
		log.Fatal(err)
	}
}

// Middleware functions
// Validate API-KEY and set userId and isadmin parameters based on key
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"result": "API key required"})
			c.Abort()
			return
		}
		user, err := database.GetUserByAPIKey(apiKey)
		if err != nil || user.Id <= 0 || !user.Enabled {
			c.JSON(http.StatusUnauthorized, gin.H{"result": "Invalid API key"})
			c.Abort()
			return
		}
		c.Set("userId", user.Id)
		c.Set("name", user.Name)
		c.Set("isadmin", user.Admin)
		c.Next()
	}
}

// Only allow request if specified tracker is owned by caller or if caller is admin
func OwnershipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		trackerId := c.Param("id")
		t, err := database.GetTracker(trackerId)
		// Allowed if Tracker exists and user is admin
		if err == nil && c.GetBool("isadmin") {
			c.Next()
			return
		}
		// Allowed if Tracker exists and user has ownership
		if c.GetInt("userId") == t.Owner {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"result": "You don't have a tracker registered with the specified id"})
		c.Abort()
	}
}

// Only allow is the user is admin
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("isadmin") {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"result": "You don't have access to this feature"})
		c.Abort()
	}
}

// Active Trackers
// Convert a model.Tracker struct to a TrackerResponse object
// This includes appending the connected state and converting the timestamp to string
func newTrackerResponse(ts []model.TrackerWithLocation) []TrackerResponse {
	active := getActiveHandlersId()

	var out []TrackerResponse
	for _, t := range ts {
		connected := false
		if slices.Contains(active, t.Tracker.Id) {
			connected = true
		}
		locationResp := LocationResponse{}
		if t.Ld != nil {
			timestamp := timeToString(t.Ld.Timestamp)
			locationResp = LocationResponse{
				Timestamp: &timestamp,
				Lat:       &t.Ld.Lat,
				Lon:       &t.Ld.Lon,
				Speed:     &t.Ld.Speed,
				Heading:   &t.Ld.Heading,
			}
		}
		out = append(out, TrackerResponse{
			Id:               t.Tracker.Id,
			Name:             t.Name,
			Owner:            t.Owner,
			PhoneNumber:      t.PhoneNumber,
			Model:            t.Model,
			Connected:        connected,
			Enabled:          t.Enabled,
			LastConnected:    timeToString(t.LastConnected),
			LocationResponse: locationResp,
		})
	}
	return out
}

// @Summary      Get list of trackers
// @Description  If the user is a admin, it will respond with a list of all trackers in the system, and if the user is a regular user, it will return all trackers owned by the user,
// @Tags         Trackers
// @Produce      json
// @Success      200  {array}   TrackerResponse
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key"
// @Router       /trackers [get]
// @Security ApiKeyAuth
func getTrackers(c *gin.Context) {
	var trackers []model.TrackerWithLocation
	if c.GetBool("isadmin") {
		trackers = database.GetTrackers()
	} else {
		trackers = database.GetTrackersByUserId(c.GetInt("userId"))
	}
	if trackers == nil {
		c.IndentedJSON(http.StatusOK, make([]model.TrackerWithLocation, 0))
		return
	}
	c.IndentedJSON(http.StatusOK, newTrackerResponse(trackers))
}

// @Summary      Get tracker by id
// @Description
// @Tags         Trackers
// @Produce      json
// @Param        id   path      string  true  "TrackerID"
// @Success      200  {object}  TrackerResponse
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR not allowed to access tracker"
// @Failure      403  {object}  StringResultRes "You don't have a tracker registered with the specified id"
// @Failure      404  {object}  StringResultRes "Tracker not found"
// @Router       /trackers/{id} [get]
// @Security     ApiKeyAuth
func getTracker(c *gin.Context) {
	id := c.Param("id")
	t, err := database.GetTracker(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"result": "Tracker not found"})
		return
	}
	// Abort if user is not admin and requesting tracker it does not own
	if t.Owner != c.GetInt("userId") && !c.GetBool("isadmin") {
		c.IndentedJSON(http.StatusForbidden, gin.H{"result": "You don't have a tracker registered with the specified id"})
		return
	}

	t_res := newTrackerResponse([]model.TrackerWithLocation{t})[0]
	c.IndentedJSON(http.StatusOK, t_res)
}

// @Summary      Register tracker
// @Description  Register tracking by supplying all properties of a tracker
// @Tags         Trackers
// @Accept       json
// @Produce      json
// @Param        body body RegisterTrackerReq true "Register tracker payload"
// @Success      201  {object}  StringResultRes "Success"
// @Failure      400  {object}  StringResultRes "failed to parse OR API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key"
// @Failure      403  {object}  StringResultRes "This action requires admin permissions"
// @Failure      409  {object}  StringResultRes "Tracker with identical id or name already exists"
// @Failure      500  {object}  StringResultRes "Failed"
// @Router       /trackers [POST]
// @Security ApiKeyAuth
func registerTracker(c *gin.Context) {
	var t RegisterTrackerReq

	if err := c.BindJSON(&t); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "failed to parse"})
		return
	}

	owner := c.GetInt("userId")
	// If owner property is set then allow if user is admin else abort
	if t.Owner != 0 {
		if c.GetBool("isadmin") {
			owner = t.Owner
		} else {
			c.IndentedJSON(http.StatusForbidden, gin.H{"result": "This action requires admin permissions"})
			log.Println("registerTracker: Parsing failed")
			return

		}
	}

	newt := model.Tracker{Id: t.Id, Name: t.Name, Owner: owner, PhoneNumber: t.PhoneNumber, Model: t.Model, Enabled: t.Enabled}
	log.Println("Trying to register:", newt)

	res := database.RegisterTracker(newt)
	if res != model.TrackerRegistrationSuccess {
		switch res {
		case model.TrackerRegistrationIdUsedByOtherUser, model.TrackerRegistrationNameUsedByOtherUser:
			c.JSON(http.StatusForbidden, gin.H{"result": ""})
		case model.TrackerRegistrationIdenticalExists, model.TrackerRegistrationIdUsedByOwner, model.TrackerRegistrationNameUsedByOwner:
			c.JSON(http.StatusConflict, gin.H{"result": "Tracker with identical id or name already exists"})
		case model.TrackerRegistrationUnknownError:
			c.JSON(http.StatusInternalServerError, gin.H{"result": "Failed"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"result": "Success"})
}

// @Summary      Deregister tracker by id
// @Description
// @Tags         Trackers
// @Produce      json
// @Param        id   path      string  true  "TrackerID"
// @Success      200  {object}  StringResultRes "success"
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR not allowed to access tracker"
// @Failure      500  {object}  StringResultRes "failed"
// @Router       /trackers/{id} [delete]
// @Security     ApiKeyAuth
func deregisterTracker(c *gin.Context) {
	id := c.Param("id")

	err := database.DeregisterTracker(id)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"result": "failed"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"result": "success"})
}

// @Summary      Enable/disable tracker
// @Description  Set the enabled state of a tracker to the provided boolean value
// @Tags         Trackers
// @Accept       json
// @Produce      json
// @Param        body body EnabledReq true "true/false"
// @Param        id   path      string  true  "TrackerID"
// @Success      201  {object}  StringResultRes "success"
// @Failure      400  {object}  StringResultRes "failed to parse OR API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR not allowed to access tracker"
// @Failure      403  {object}  StringResultRes "This action requires admin permissions"
// @Failure      500  {object}  StringResultRes "failed"
// @Router       /trackers/{id}/enabled [PUT]
// @Security ApiKeyAuth

func setEnabled(c *gin.Context) {
	id := c.Param("id")
	var er EnableReq
	if err := c.BindJSON(&er); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "failed to parse"})
		return
	}
	err := database.SetTrackerEnabled(id, er.Enabled)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"result": "failed"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"result": "success"})
}

// @Summary      Get list of tracker models
// @Description  Get a list of all tracker models currently supported
// @Tags         Models
// @Produce      json
// @Success      200  {array}   model.Model
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key"
// @Router       /models [get]
// @Security ApiKeyAuth
func getModels(c *gin.Context) {
	models := database.GetModels()
	if models == nil {
		c.IndentedJSON(http.StatusOK, make([]model.Model, 0))
		return
	}
	c.IndentedJSON(http.StatusOK, models)
}

// @Summary      Get model by name
// @Description
// @Tags         Models
// @Produce      json
// @Param        name   path      string  true  "Model name"
// @Success      200  {object}  model.Model
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key"
// @Failure      404  {object}  StringResultRes "Model was not found"
// @Router       /models/{name} [get]
// @Security     ApiKeyAuth
func getModel(c *gin.Context) {
	m, err := database.GetModel(c.Param("name"))
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"result": "Model was not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, m)

}

// @Summary      Create model
// @Description  Add support for new model, by specifing which SMS messages should be sent when the tracker model is provisioned. The tracker model, must support GT06 or JT808, to work.
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        body body CreateModelReq true "Register model payload"
// @Success      200  {object}  StringResultRes "success"
// @Failure      400  {object}  StringResultRes "failed to parse OR API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR You don't have access to this feature"
// @Failure      500  {object}  StringResultRes "failed"
// @Router       /models [POST]
// @Security ApiKeyAuth
func createModel(c *gin.Context) {
	var m CreateModelReq

	if err := c.BindJSON(&m); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "failed to parse"})
		return
	}

	newm := model.Model{Name: m.Name, Init_commands: m.Init_commands, Success_keywords: m.Success_keywords}
	log.Println("Trying to register:", newm)

	if err := database.CreateModel(newm); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"result": "failed"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"result": "success"})

}

// @Summary      Delete model
// @Description  Remove support for tracker model
// @Tags         Models
// @Produce      json
// @Param        name   path      string  true  "Model name"
// @Success      200  {object}  StringResultRes "success"
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR You don't have access to this feature"
// @Failure      500  {object}  StringResultRes "failed"
// @Router       /models/{name} [delete]
// @Security     ApiKeyAuth
func deleteModel(c *gin.Context) {
	name := c.Param("name")
	err := database.DeleteModel(name)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "failed"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "success"})
}

// @Summary      Get tracker location
// @Description  Get the latest location data event reported by specified tracker
// @Tags         Location
// @Produce      json
// @Param        id   path      string  true  "TrackerID"
// @Success      200  {object}  LocationResponse
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR not allowed to access tracker"
// @Failure      404  {object}  StringResultRes "No location entry found"
// @Router       /trackers/{id}/location [get]
// @Security     ApiKeyAuth
func getTrackerLocation(c *gin.Context) {
	id := c.Param("id")
	ld, err := database.GetLocation(id)
	tm := timeToString(ld.Timestamp)
	lr := LocationResponse{Timestamp: &tm, Lat: &ld.Lat, Lon: &ld.Lon, Speed: &ld.Speed, Heading: &ld.Heading}
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"result": "No location entry found"})
		return
	}
	c.IndentedJSON(http.StatusOK, lr)
}

// @Summary      Get tracker locations
// @Description  Get a array with all location data events reported by specified tracker
// @Tags         Location
// @Produce      json
// @Param        id   path      string  true  "TrackerID"
// @Success      200  {array}  []LocationResponse
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR not allowed to access tracker"
// @Failure      404  {object}  StringResultRes "No location entry found"
// @Router       /trackers/{id}/locations [get]
// @Security     ApiKeyAuth
func getTrackerLocations(c *gin.Context) {
	id := c.Param("id")
	// Optional query parameters: ?limit=N or ?start=RFC3339|unix&end=RFC3339|unix
	// If start/end are provided, they take precedence over limit.
	startQ := c.Query("start")
	endQ := c.Query("end")
	if startQ != "" || endQ != "" {
		// Parse start and end
		var start, end int64
		var err error
		// default end to now if missing
		if endQ == "" {
			end = time.Now().Unix()
		} else {
			end, err = parseTimeQuery(endQ)
			if err != nil {
				c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "invalid end parameter"})
				return
			}
		}
		if startQ == "" {
			// default start to 24 hours before end if missing
			start = end - 24*3600
		} else {
			start, err = parseTimeQuery(startQ)
			if err != nil {
				c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "invalid start parameter"})
				return
			}
		}
		ld, err := database.GetTrackerLocationHistoryRange(id, start, end)
		if err != nil || len(ld) == 0 {
			c.IndentedJSON(http.StatusNotFound, gin.H{"result": "No location entry found"})
			return
		}
		lh := make([]LocationResponse, len(ld))
		for i := 0; i < len(ld); i++ {
			tm := timeToString(ld[i].Timestamp)
			lh[i] = LocationResponse{Timestamp: &tm, Lat: &ld[i].Lat, Lon: &ld[i].Lon, Speed: &ld[i].Speed, Heading: &ld[i].Heading}
		}
		c.IndentedJSON(http.StatusOK, lh)
		return
	}

	// Optional query parameter: ?limit=N to fetch only the last N locations
	limitStr := c.Query("limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "invalid limit parameter"})
			return
		}
		ld, err := database.GetTrackerLocationHistoryLimit(id, limit)
		if err != nil {
			c.IndentedJSON(http.StatusNotFound, gin.H{"result": "No location entry found"})
			return
		}
		// DB returns latest-first (DESC). Return chronological order (oldest -> newest) to callers.
		for i, j := 0, len(ld)-1; i < j; i, j = i+1, j-1 {
			ld[i], ld[j] = ld[j], ld[i]
		}
		lh := make([]LocationResponse, len(ld))
		for i := 0; i < len(ld); i++ {
			tm := timeToString(ld[i].Timestamp)
			lh[i] = LocationResponse{Timestamp: &tm, Lat: &ld[i].Lat, Lon: &ld[i].Lon, Speed: &ld[i].Speed, Heading: &ld[i].Heading}
		}
		c.IndentedJSON(http.StatusOK, lh)
		return
	}

	// No limit provided: return full history
	ld, err := database.GetTrackerLocationHistory(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"result": "No location entry found"})
		return
	}
	lh := make([]LocationResponse, len(ld))
	for i := 0; i < len(ld); i++ {
		tm := timeToString(ld[i].Timestamp)
		lh[i] = LocationResponse{Timestamp: &tm, Lat: &ld[i].Lat, Lon: &ld[i].Lon, Speed: &ld[i].Speed, Heading: &ld[i].Heading}
	}
	c.IndentedJSON(http.StatusOK, lh)
}

// @Summary      Send command
// @Description  Send upstream command to specified tracker, and get tracker response. The request will fail if the tracker is not currently connected. Additionally the request may timeout, if the tracker is connected but does not response. This can happen if the tracker has entered sleep mode without first closing the TCP connection
// @Tags         Commands
// @Produce      json
// @Accept       json
// @Param        id   path      string  true  "TrackerID"
// @Param        body   body CommandReq  true  "Command"
// @Success      200  {object}  StringResultRes "RESPONSE"
// @Failure      400  {object}  StringResultRes "failed to parse OR API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key OR not allowed to access tracker"
// @Failure      503  {object}  StringResultRes "The tracker is not connected"
// @Router       /trackers/{id}/command [post]
// @Security     ApiKeyAuth
func sendCommand(c *gin.Context) {
	var cmd CommandReq
	id := c.Param("id")

	active := getActiveHandlersId()
	// Abort if tracker is not connected
	if !slices.Contains(active, id) {
		c.IndentedJSON(http.StatusServiceUnavailable, gin.H{"result": "The tracker is not connected"})
		return
	}
	if err := c.BindJSON(&cmd); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"result": "failed to parse"})
		return
	}
	// Create response channel
	resChannel := make(chan string)
	// Create TrackerCommand object and pass to command queue
	tc := model.TrackerCommand{TrackerId: id, Payload: cmd.Command, Response: resChannel}
	tm.CommandQueue <- tc
	// Wait for response
	res := <-resChannel
	// Pass response to API caller
	c.IndentedJSON(http.StatusOK, gin.H{"result": res})
}

// @Summary      Whoami
// @Description  Fetch the user/organization name associated with the used API key. This can be used to detect if a api-key is valid
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  NameRes "NAME OF USER/ORGANISATION"
// @Failure      400  {object}  StringResultRes "API key required"
// @Failure      401  {object}  StringResultRes "Invalid API key"
// @Router       /whoami [get]
// @Security     ApiKeyAuth
func whoami(c *gin.Context) {
	name := c.GetString("name")
	c.IndentedJSON(http.StatusOK, gin.H{"name": name})
}

// Misc
func timeToString(t int64) string {
	return time.Unix(t, 0).Format(time.RFC3339)
}

func getActiveHandlersId() []string {
	tm.Mu.Lock()
	keys := maps.Keys(tm.Handlers)
	tm.Mu.Unlock()
	return slices.Collect(keys)
}

// parseTimeQuery parses either RFC3339 string or unix seconds into epoch seconds
func parseTimeQuery(v string) (int64, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.Unix(), nil
	}
	// Fallback: int64 unix seconds
	if sec, err := strconv.ParseInt(v, 10, 64); err == nil {
		return sec, nil
	}
	return 0, fmt.Errorf("invalid time")
}
