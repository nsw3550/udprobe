package udprobe

import (
	"net"
	"testing"
	"time"
)

var exampleProbe = Probe{
	Pd:    &PathDist{},
	CSent: uint64(1234123412),
	CRcvd: uint64(1234567890),
	Tos:   byte(0),
}

var exampleUDPAddr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:0")
var exampleUDPAddrChan = make(chan *net.UDPAddr)
var exampleBoolChan = make(chan bool)
var exampleProbeChan = make(chan *Probe)

/*
   Port tests
*/
func TestSrcPD(t *testing.T) {
	// TODO(nwinemiller): This will need some mocking in order to be build safe.
}

func TestPd(t *testing.T) {
	// TODO(nwinemiller): This will need some mocking in order to be build safe.
}

func TestTos(t *testing.T) {
	// This really is just a helper for `GetTos` and doesn't need testing
}

func TestSend(t *testing.T) {
	// TODO(nwinemiller): This will need some mocking in order to be build safe.
}

func TestRecv(t *testing.T) {
	// TODO(nwinemiller): This will need some mocking in order to be build safe.
}

func TestDone(t *testing.T) {
	// This is basically just IfaceToProbe and passing to a channel, so
	// doesn't really need testing.
}

func TestNewPort(t *testing.T) {
	// Just test creating one
	conn, _ := net.ListenUDP("udp", exampleUDPAddr)
	_ = NewPort(
		conn,
		exampleUDPAddrChan,
		exampleBoolChan,
		exampleProbeChan,
		time.Second,
		3*time.Second,
		200*time.Millisecond,
	)
}

func TestNewDefault(t *testing.T) {
	// Just test creating one
	_ = NewDefault(
		exampleUDPAddrChan,
		exampleBoolChan,
		exampleProbeChan,
	)
}

/*
   End Port tests
*/

func TestSendValidation(t *testing.T) {
	tosend := make(chan *net.UDPAddr)
	stop := make(chan bool)
	cbc := make(chan *Probe)

	// Create a default UDPConn
	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, _ := net.ListenUDP("udp", udpAddr)
	defer conn.Close()

	port := NewPort(
		conn,
		tosend,
		stop,
		cbc,
		time.Second,
		3*time.Second,
		200*time.Millisecond,
	)

	go port.send()
	defer func() { stop <- true }()

	// 1. Test nil IP
	nilAddr := &net.UDPAddr{Port: 1234, IP: nil}
	tosend <- nilAddr

	// Give it a moment to process (or skip)
	time.Sleep(10 * time.Millisecond)

	if port.cache.Len() != 0 {
		t.Errorf("Expected cache to be empty after sending nil IP, but got %d items", port.cache.Len())
	}

	// 2. Test valid IP
	validAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:1234")
	tosend <- validAddr

	// Give it a moment to process
	time.Sleep(50 * time.Millisecond)

	if port.cache.Len() != 1 {
		t.Errorf("Expected cache to have 1 item after sending valid IP, but got %d items", port.cache.Len())
	}
}

func TestIfaceToProbe(t *testing.T) {
	// Convert the example
	converted, err := IfaceToProbe(&exampleProbe)
	if err != nil {
		t.Error("Encountered an error when converting to probe")
	}
	// Make sure it matches the original
	if &exampleProbe != converted {
		t.Error("Converted to probe, but doesn't match original")
	}
	// Make sure passing in something else fails
	_, err = IfaceToProbe("I am not a probe")
	if err == nil {
		t.Error("Expected an error current conversion, but didn't get one")
	}
}
