package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

var routeRegStart = time.Now()

var (
	processStartTime time.Time
	serverReadyTime  time.Time
	firstRequestTime time.Time
	routeRegTime     time.Duration
	serverStartTime  time.Duration
)

type TimingResponse struct {
	Status           string `json:"status"`
	ProcessStartTime string `json:"process_start_time"`
	RouteRegTime     string `json:"route_registration_time"`
	ServerStartTime  string `json:"server_start_time"`
	ServerReadyTime  string `json:"server_ready_time"`
	FirstRequestTime string `json:"first_request_time,omitempty"`
	TotalStartup     string `json:"total_startup_time"`
	Uptime           string `json:"uptime"`
}

func init() {
	processStartTime = time.Now()
}

func timingHandler(w http.ResponseWriter, r *http.Request) {
	if firstRequestTime.IsZero() {
		firstRequestTime = time.Now()
	}

	response := TimingResponse{
		Status:           "ok",
		ProcessStartTime: processStartTime.Format(time.RFC3339Nano),
		RouteRegTime:     routeRegTime.String(),
		ServerStartTime:  serverStartTime.String(),
		ServerReadyTime:  serverReadyTime.Format(time.RFC3339Nano),
		TotalStartup:     time.Since(processStartTime).String(),
		Uptime:           time.Since(serverReadyTime).String(),
	}

	if !firstRequestTime.IsZero() {
		response.FirstRequestTime = firstRequestTime.Format(time.RFC3339Nano)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if firstRequestTime.IsZero() {
		firstRequestTime = time.Now()
	}

	// Include timing info in health response
	uptime := time.Since(serverReadyTime)
	response := map[string]interface{}{
		"status":            "ok",
		"uptime":            uptime.String(),
		"server_start_time": serverStartTime.String(),
		"route_reg_time":    routeRegTime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Register routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/timing", timingHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Self-timing server","endpoints":["/health","/timing"]}`))
	})

	routeRegTime = time.Since(routeRegStart)

	fmt.Printf("⏱️  Process started at: %v\n", processStartTime.Format("15:04:05.000000"))
	fmt.Printf("⏱️  Route registration took: %v\n", routeRegTime)
	fmt.Printf("🚀 Starting HTTP server on :8080\n")

	// Measure actual server startup time
	serverStartStart := time.Now()

	// Create server with custom listener to detect when it's actually ready
	server := &http.Server{
		Addr: ":8080",
	}

	// Listen first to get the actual port binding time
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		fmt.Printf("❌ Failed to bind to port: %v\n", err)
		return
	}

	// Server is now ready to accept connections
	serverReadyTime = time.Now()
	serverStartTime = time.Since(serverStartStart)

	fmt.Printf("✅ Server ready at: %v (took %v)\n",
		serverReadyTime.Format("15:04:05.000000"),
		serverStartTime)
	fmt.Printf("📊 Total startup time: %v\n", time.Since(processStartTime))
	fmt.Printf("🎯 Actual server binding time: %v\n", serverStartTime)

	// Now serve using the pre-bound listener
	if err := server.Serve(listener); err != nil {
		fmt.Printf("❌ Server failed: %v\n", err)
	}
}
