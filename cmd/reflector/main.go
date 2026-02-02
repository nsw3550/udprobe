package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"github.com/nsw3550/llama"
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
	llama.HandleError(err)

	// Create a connection at the local address which is used for listening
	conn, err := net.ListenUDP("udp", myAddr)
	llama.HandleError(err)
	// Cleanup after
	defer func(c *net.UDPConn) {
		err := c.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(conn)

	// Tell the socket to get timestamps and increase buffer size
	// NOTE(dmar): We aren't actually using the socket timestamps yet
	llama.EnableTimestamps(conn)
	llama.SetRecvBufferSize(conn, BUFFER_SIZE)

	// Create the rate limiter to be used in the reflector
	// NOTE(dmar): This has the potential to be spikey if there are gaps between
	//     processing periods. So it's somewhat reliant on a smooth stream of
	//     incoming probes.
	rateLimiter := rate.NewLimiter(rate.Limit(*maxPPS), int(*maxPPS))

	// Start API server if enabled
	var api *llama.ReflectorAPI
	if !*noAPI {
		api = llama.NewReflectorAPI(*apiBind)
		api.Run()
		log.Printf("API listening on %s", *apiBind)
	}

	// Begin reflecting
	llama.Reflect(conn, rateLimiter)
}
