package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/nsw3550/udprobe"
	"golang.org/x/sys/unix"
)

func main() {
	flag.Parse()

	// Create the collector
	collector := udprobe.Collector{}

	// Perform setup
	collector.Setup()

	// Let's do this
	collector.Run()

	// Handle signals for stopping, or reloading the config and updating things
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM, unix.SIGHUP)
	for {
		sig := <-sigChan
		switch sig {
		case unix.SIGINT, unix.SIGTERM:
			udprobe.LogInfo("Received signal, shutting down")
			// TODO(nwinemiller): Add smarter handling here for around stopping things
			return
		case unix.SIGHUP:
			udprobe.LogInfo("Received SIGHUP, reloading and reconfiguring")
			collector.Reload()
		}
	}
}
