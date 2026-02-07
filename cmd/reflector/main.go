package main

import (
	"flag"
	"net"
	"strconv"

	"github.com/nsw3550/udprobe"
	"golang.org/x/time/rate"
)

var port = flag.Int("port", 8100, "Port to listen on for probes")

// If this rate is exceeded, buffering will occur, and latency will
// be impacted. If severe enough, there's a possibility of drops.
// This exists to limit the reflector's ability to utilize CPU resources.
var maxPPS = flag.Float64("max-pps", 5000, "Rate limit on packets per second")

// API server address for metrics/health (default: 8200 to avoid conflicts with node_exporter)
var apiBind = flag.String("api-bind", ":8200", "API server address for metrics/health")

// Disable HTTP API server for metrics and health checks
var noAPI = flag.Bool("no-api", false, "Disable HTTP API server")

// 540672 bytes = 528KB
var BUFFER_SIZE int = 540672

func main() {
	// Get command line args
	flag.Parse()

	// Get the localhost address specified
	myAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(*port))
	udprobe.HandleError(err)

	// Create a connection at the local address which is used for listening
	conn, err := net.ListenUDP("udp", myAddr)
	udprobe.HandleError(err)
	// Cleanup after
	defer func(c *net.UDPConn) {
		err := c.Close()
		if err != nil {
			udprobe.HandleFatalErrorMsg(err, "failed to close connection")
		}
	}(conn)

	// Tell the socket to get timestamps and increase buffer size
	// NOTE(nwinemiller): We aren't actually using the socket timestamps yet
	udprobe.EnableTimestamps(conn)
	udprobe.SetRecvBufferSize(conn, BUFFER_SIZE)

	// Create the rate limiter to be used in the reflector
	// NOTE(nwinemiller): This has the potential to be spikey if there are gaps between
	//     processing periods. So it's somewhat reliant on a smooth stream of
	//     incoming probes.
	rateLimiter := rate.NewLimiter(rate.Limit(*maxPPS), int(*maxPPS))

	// Start API server if enabled
	var api *udprobe.ReflectorAPI
	if !*noAPI {
		api = udprobe.NewReflectorAPI(*apiBind)
		api.Run()
		udprobe.LogInfo("API listening on " + *apiBind)
	}

	// Begin reflecting
	udprobe.Reflect(conn, rateLimiter)
}
