package udprobe

import (
	"net"
	"testing"
	"time"
)

var exampleProbe = InFlightProbe{
	Pd:    &PathDist{},
	CSent: uint64(1234123412),
	CRcvd: uint64(1234567890),
	Tos:   byte(0),
}

var exampleUDPAddr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:0")
var exampleUDPAddrChan = make(chan *net.UDPAddr)
var exampleBoolChan = make(chan bool)
var exampleProbeChan = make(chan *InFlightProbe)

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
	cbc := make(chan *InFlightProbe)

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
	// Close, rather than send, so the send loop and the stop watcher both
	// observe the stop signal.
	defer close(stop)

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

func TestIfaceToInFlightProbe(t *testing.T) {
	// Convert the example
	converted, err := IfaceToInFlightProbe(&exampleProbe)
	if err != nil {
		t.Error("Encountered an error when converting to InFlightProbe")
	}
	// Make sure it matches the original
	if &exampleProbe != converted {
		t.Error("Converted to InFlightProbe, but doesn't match original")
	}
	// Make sure passing in something else fails
	_, err = IfaceToInFlightProbe("I am not a InFlightProbe")
	if err == nil {
		t.Error("Expected an error current conversion, but didn't get one")
	}
}

func TestRecvStopsOnStop(t *testing.T) {
	// Regression test for the recv() hang: on stop, recv must exit
	// promptly even with zero traffic. The stop watcher closes the conn,
	// which force-unblocks any in-progress read, so a clean exit within a
	// few seconds of stop is the expected behavior. (Previously, a wedged
	// ReadMsgUDP could hang forever, ignoring the read deadline.)
	tosend := make(chan *net.UDPAddr)
	stop := make(chan bool)
	cbc := make(chan *InFlightProbe)

	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, _ := net.ListenUDP("udp", udpAddr)

	port := NewPort(
		conn,
		tosend,
		stop,
		cbc,
		time.Second,
		3*time.Second,
		200*time.Millisecond,
	)

	// Start the real recv loop and track when it exits
	recvDone := make(chan struct{})
	go func() {
		port.recv()
		close(recvDone)
	}()

	// Let recv get into its read loop
	time.Sleep(100 * time.Millisecond)

	close(stop)

	select {
	case <-recvDone:
		// recv exited cleanly after stop
	case <-time.After(5 * time.Second):
		t.Fatal("recv did not exit within 5s of stop - " +
			"the read appears wedged (deadline ignored)")
	}
}

func TestSendStopsCleanly(t *testing.T) {
	// After stop, send must exit cleanly. The stop watcher closes the
	// conn, so a subsequent WriteToUDP may hit a closed conn. That must
	// result in a clean exit, not a process kill (it previously called
	// HandleError -> os.Exit(1) on a closed conn).
	tosend := make(chan *net.UDPAddr)
	stop := make(chan bool)
	cbc := make(chan *InFlightProbe)

	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, _ := net.ListenUDP("udp", udpAddr)

	port := NewPort(
		conn,
		tosend,
		stop,
		cbc,
		time.Second,
		3*time.Second,
		200*time.Millisecond,
	)

	sendDone := make(chan struct{})
	go func() {
		port.send()
		close(sendDone)
	}()

	// Let send get into its select loop
	time.Sleep(100 * time.Millisecond)

	close(stop)
	// Wait for the stop watcher to close the conn
	time.Sleep(100 * time.Millisecond)

	// Push a target through after stop: send may not have observed the
	// stop chan yet, so the write could hit a closed conn. Either way,
	// send must exit promptly and the process must survive.
	target, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	select {
	case tosend <- target:
	case <-time.After(time.Second):
		// send already exited; that's fine
	}

	select {
	case <-sendDone:
		// send exited cleanly
	case <-time.After(5 * time.Second):
		t.Fatal("send did not exit within 5s of stop")
	}
}
