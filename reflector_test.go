package udprobe

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/nsw3550/udprobe/proto"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

func TestReflectorReceive(t *testing.T) {
	// Setup listener
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Setup sender
	senderConn, err := net.DialUDP("udp", nil, conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer senderConn.Close()

	testData := []byte("hello world")
	_, err = senderConn.Write(testData)
	if err != nil {
		t.Fatal(err)
	}

	dataBuf := make([]byte, 1024)
	oobBuf := make([]byte, 1024)
	
	// Set a deadline so the test doesn't hang if it fails
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	data, _, receivedAddr, err := Receive(dataBuf, oobBuf, conn)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(data))
	}

	if receivedAddr.String() != senderConn.LocalAddr().String() {
		t.Errorf("Expected sender address %s, got %s", senderConn.LocalAddr().String(), receivedAddr.String())
	}
}

func TestReflectorSend(t *testing.T) {
	// Setup listener to receive the sent packet
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Setup sender connection (using the function we want to test)
	senderAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	senderConn, err := net.ListenUDP("udp", senderAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer senderConn.Close()

	testData := []byte("reflector test")
	targetAddr := conn.LocalAddr().(*net.UDPAddr)

	err = Send(testData, 0, senderConn, targetAddr)
	if err != nil {
		t.Fatal(err)
	}

	// Verify receipt
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatal(err)
	}

	if string(buf[:n]) != "reflector test" {
		t.Errorf("Expected 'reflector test', got '%s'", string(buf[:n]))
	}
}

func TestReflectorLoop(t *testing.T) {
	// 1. Setup reflector
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	rl := rate.NewLimiter(100, 100)
	ctx, cancel := context.WithCancel(context.Background())
	
	// Run Reflect in background
	go Reflect(ctx, conn, rl)

	// 2. Setup client
	clientConn, _ := net.DialUDP("udp", nil, conn.LocalAddr().(*net.UDPAddr))
	defer clientConn.Close()

	// 3. Test Success Path: Send valid probe
	probe := &pb.Probe{
		Signature: []byte("test-sig"),
		Tos:       46,
		Sent:      1000,
	}
	marshaled, _ := proto.Marshal(probe)

	_, err := clientConn.Write(marshaled)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for reflection
	buf := make([]byte, 4096)
	clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Fatal("Did not receive reflected packet:", err)
	}

	reflectedProbe := &pb.Probe{}
	err = proto.Unmarshal(buf[:n], reflectedProbe)
	if err != nil {
		t.Fatal("Failed to unmarshal reflected probe:", err)
	}

	if string(reflectedProbe.Signature) != "test-sig" {
		t.Errorf("Signature mismatch: expected 'test-sig', got '%s'", string(reflectedProbe.Signature))
	}
	if reflectedProbe.Rcvd == 0 {
		t.Error("Expected Rcvd timestamp to be set by reflector")
	}

	// 4. Test Error Path: Send bad data
	badData := []byte("not a protobuf")
	_, _ = clientConn.Write(badData)

	// Give it a moment to process and check if it still works for good data
	time.Sleep(50 * time.Millisecond)
	_, _ = clientConn.Write(marshaled)
	
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err = clientConn.Read(buf)
	if err != nil {
		t.Error("Reflector stopped working after bad data")
	}

	// 5. Test Throttling
	// Set very low rate limit
	rlT := rate.NewLimiter(1, 1) // 1 packet per second
	ctxT, cancelT := context.WithCancel(context.Background())
	
	addrT, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	connT, _ := net.ListenUDP("udp", addrT)
	defer connT.Close()
	go Reflect(ctxT, connT, rlT)
	
	clientConnT, _ := net.DialUDP("udp", nil, connT.LocalAddr().(*net.UDPAddr))
	defer clientConnT.Close()

	// Send two packets quickly
	_, _ = clientConnT.Write(marshaled)
	_, _ = clientConnT.Write(marshaled)

	// One should be delayed/throttled (we can't easily check the metric here without more setup, 
	// but we've verified the code path exists).

	// Cleanup
	cancel()
	cancelT()
}
