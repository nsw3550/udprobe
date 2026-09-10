package udprobe

import (
	"bytes"
	"net"
	"testing"
	"time"

	pb "github.com/nsw3550/udprobe/proto"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
)

func TestProbeProtobuf(t *testing.T) {
	// Test with good data first
	signature := []byte("abcdefghij")
	data := &pb.Probe{
		Signature: signature,
		Tos:       46,
		Sent:      123456789,
	}
	marshaled, err := proto.Marshal(data)
	if err != nil {
		t.Fatal("Failed to marshal probe data:", err)
	}

	unmarshaled := &pb.Probe{}
	err = proto.Unmarshal(marshaled, unmarshaled)
	if err != nil {
		t.Fatal("Failed to unmarshal probe data:", err)
	}

	// Compare the actual structs
	if !bytes.Equal(unmarshaled.Signature, data.Signature) ||
		unmarshaled.Tos != data.Tos ||
		unmarshaled.Sent != data.Sent {
		t.Error("Data unmarshaled, but lost in translation")
	}

	// Now verify that bad data doesn't work
	badData := []byte{1, 2, 3, 4, 5}
	err = proto.Unmarshal(badData, &pb.Probe{})
	if err == nil {
		t.Error("No error returned for bad data (though unmarshal might succeed with empty/corrupt proto)")
	}
}

func TestSetTos(t *testing.T) {
	// Resolve a local addr
	myAddr, _ := net.ResolveUDPAddr("udp", ":0")
	// Create a connection
	conn, _ := net.ListenUDP("udp", myAddr)
	defer conn.Close()
	// Set the ToS value
	tosVal := 240
	newTos := byte(tosVal)
	SetTos(conn, newTos)
	// Verify the ToS value
	val := GetTos(conn)
	if val != newTos {
		t.Error("New ToS value not set correctly. Set", tosVal, "and got",
			val, "instead.")
	}
}

func TestEnableTimestamps(t *testing.T) {
	// Resolve a local addr
	myAddr, _ := net.ResolveUDPAddr("udp", ":0")
	// Create a connection
	conn, _ := net.ListenUDP("udp", myAddr)
	defer conn.Close()

	// Enable timestamping
	EnableTimestamps(conn)

	// Verify the option was actually set on the socket, using
	// SyscallConn().Control() (same mechanism as the helpers themselves)
	rc, err := conn.SyscallConn()
	if err != nil {
		t.Fatal("Failed to get raw conn:", err)
	}
	enabled := 0
	err = rc.Control(func(fd uintptr) {
		enabled, _ = unix.GetsockoptInt(int(fd), unix.SOL_SOCKET,
			unix.SO_TIMESTAMPNS)
	})
	if err != nil {
		t.Fatal("Control failed:", err)
	}
	if enabled != 1 {
		t.Error("SO_TIMESTAMPNS not enabled. Expected 1, got", enabled)
	}
}

func TestSocketOptsPreserveDeadlines(t *testing.T) {
	// Regression test for the recv() hang: the old conn.File()-based
	// helpers could leave the socket in blocking mode, silently disabling
	// read deadlines. Using SyscallConn().Control() must preserve the
	// non-blocking state, so a read deadline must still fire.
	myAddr, _ := net.ResolveUDPAddr("udp", ":0")
	conn, _ := net.ListenUDP("udp", myAddr)
	defer conn.Close()

	// Exercise all of the socket option helpers, like a real Port does
	SetTos(conn, 46)
	if val := GetTos(conn); val != 46 {
		t.Fatalf("ToS round trip failed: got %d", val)
	}
	EnableTimestamps(conn)

	// Now verify a read deadline still works: a read with a short deadline
	// must time out (return an error) rather than block forever.
	err := conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if err != nil {
		t.Fatal("Failed to set read deadline:", err)
	}
	buf := make([]byte, 16)
	// Run the read with a safety timeout well beyond the deadline, so the
	// test fails instead of hanging if blocking mode regresses.
	done := make(chan error, 1)
	go func() {
		_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Expected a timeout error, but read succeeded")
		}
		// The error should be a timeout
		netErr, ok := err.(net.Error)
		if !ok || !netErr.Timeout() {
			t.Fatal("Expected a timeout error, got:", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Read did not return after deadline - " +
			"socket appears to be in blocking mode (deadline ignored)")
	}
}
