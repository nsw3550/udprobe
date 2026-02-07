package udprobe

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

const (
	// Listens on any addr to an automatically assigned port number
	DefaultAddrStr        = "0.0.0.0:0"
	DefaultTos            = byte(0)
	DefaultRcvBuff        = 2097600 // 2MiB
	DefaultReadTimeout    = 200 * time.Millisecond
	DefaultCacheTimeout   = 2 * time.Second
	DefaultCacheCleanRate = 5 * time.Second
	ExpireNow             = time.Nanosecond
)

// NewID returns 10 bytes of a new UUID4 as a string.
//
// This should be unique enough for short-lived cases, but as it's only a
// partial UUID.
func NewID() string {
	fullUUID := uuid.New()
	last10 := fullUUID[len(fullUUID)-10:]
	return string(last10)
}

// IDTo10Bytes converts a string to a 10 byte array.
func IDToBytes(id string) [10]byte {
	var arr [10]byte
	copy(arr[:], id)
	return arr
}

// NowUint64 returns the current time in nanoseconds as a uint64.
func NowUint64() uint64 {
	return uint64(time.Now().UnixNano())
}

// FileCloseHandler will close an open File and handle the resulting error.
func FileCloseHandler(f *os.File) {
	// NOTE: This is required, specifically for sockets/net.Conn because it
	// would appear that calls like setting the ToS value or enabling
	// timestamps cause this to go into a blocking state. Which then disables
	// the functionality of SetReadDeadline, making reads block infinitely.
	err := unix.SetNonblock(int(f.Fd()), true)
	HandleError(err)
	err = f.Close()
	HandleError(err)
}

func HandleError(err error) {
	if err != nil {
		log.Output(2, "FATAL ERROR: "+err.Error())
		os.Exit(1)
	}
}

func HandleMinorError(err error) {
	if err != nil {
		log.Output(2, "MINOR ERROR: "+err.Error())
	}
}

func HandleMinorErrorMsg(err error, msg string) {
	if err != nil {
		log.Output(2, "MINOR ERROR: "+msg+": "+err.Error())
	}
}

// HandleFatalError receives an error, then logs and exits if not nil.
func HandleFatalError(err error) {
	if err != nil {
		log.Output(2, "FATAL ERROR: "+err.Error())
		os.Exit(1)
	}
}

func HandleFatalErrorMsg(err error, msg string) {
	if err != nil {
		log.Output(2, "FATAL ERROR: "+msg+": "+err.Error())
		os.Exit(1)
	}
}

func LogWarning(msg string) {
	log.Output(2, "WARNING: "+msg)
}

func LogInfo(msg string) {
	log.Output(2, "INFO: "+msg)
}

// SetRecvBufferSize sets the size of the receive buffer for the conn to the
// provided size in bytes.
func SetRecvBufferSize(conn *net.UDPConn, size int) {
	err := conn.SetReadBuffer(size)
	HandleError(err)
}
