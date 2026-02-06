package udprobe

import (
	"net"
	"testing"
)

func TestNewID(t *testing.T) {
	// Make sure it's only a length of 10 bytes
	id := NewID()
	if len(id) != 10 {
		t.Error("NewID return more or less than 10 bytes")
	}
	// Make sure it creates new ones each time
	id2 := NewID()
	if id == id2 {
		t.Error("NewID returning duplicates")
	}
}

func TestIDToBytes(t *testing.T) {
	// Simply running this should do the trick. It's all type checked.
	str := "abcdefghij"
	bytes := [10]byte{97, 98, 99, 100, 101, 102, 103, 104, 105, 106}
	converted := IDToBytes(str)
	if converted != bytes {
		t.Error("IDToBytes failed to convert correctly")
	}
}

func TestNowUint64(t *testing.T) {
	// Just make sure it's a greater value each time
	val := NowUint64()
	for i := 0; i <= 10; i++ {
		newVal := NowUint64()
		if newVal <= val {
			t.Error("NowUint64 doesn't appear to be incrementing")
		}
		val = newVal
	}
}

func TestSetRecvBufferSize(t *testing.T) {
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	size := 1024 * 1024 // 1MiB
	SetRecvBufferSize(conn, size)

	// We can't easily verify the size without using syscalls or conn.File(),
	// but at least we know it didn't panic or fail (HandleError would have exited).
}
