package udprobe

import (
	"net"
	"time"
	"unsafe"

	pb "github.com/nsw3550/udprobe/proto"
	"golang.org/x/sys/unix"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

// Reflect will listen on the provided UDPConn and will send back any UdpData
// compliant packets that it receives, in compliance with the RateLimiter.
func Reflect(conn *net.UDPConn, rl *rate.Limiter) {
	reflectorUp.Set(1)
	defer reflectorUp.Set(0)

	dataBuf := make([]byte, 4096)
	oobBuf := make([]byte, 4096)

	LogInfo("Beginning reflection on: " + conn.LocalAddr().String())
	for {
		// Use reserve so we can track when throttling happens
		reservation := rl.Reserve()
		delay := reservation.Delay()
		if delay > 0 {
			// We hit the rate limit, so log it
			time.Sleep(delay)
			reflectorPacketsThrottled.Inc()
		}

		// Receive data from the connection
		// Not currently using `oob`
		reflectorPacketsReceived.Inc()
		data, _, addr := Receive(dataBuf, oobBuf, conn)

		// For this section, it might make sense to put in `Process` anyways.
		// But for now, all we need is to make sure it's udprobe data
		// and get the ToS value.
		pbProbe := &pb.Probe{}
		err := proto.Unmarshal(data, pbProbe)
		if err != nil {
			// Else, don't reflect bad data
			reflectorPacketsBadData.Inc()
			HandleMinorErrorMsg(err, "failed to unmarshal probe")
			continue
		}

		// Send the data back to sender
		Send(data, pbProbe.Tos[0], conn, addr)
		reflectorPacketsReflected.Inc()
	}
}

// Receive accepts UDP packets on the provided conn and returns the data and
// and control message slices, as well as the UDPAddr it was received from.
func Receive(data []byte, oob []byte, conn *net.UDPConn) (
	[]byte, []byte, *net.UDPAddr,
) {
	// Receive the data from the connection
	dataLen, oobLen, _, addr, err := conn.ReadMsgUDP(data, oob)
	HandleError(err)
	return data[0:dataLen], oob[0:oobLen], addr
}

// Send will send the provided data using the conn to the addr, via UDP.
func Send(data []byte, tos byte, conn *net.UDPConn, addr *net.UDPAddr) {
	oob := make([]byte, unix.CmsgSpace(1))
	h := (*unix.Cmsghdr)(unsafe.Pointer(&oob[0]))
	h.Level = unix.IPPROTO_IP
	h.Type = unix.IP_TOS
	h.SetLen(unix.CmsgLen(1))
	dataPtr := uintptr(unsafe.Pointer(h)) + uintptr(unix.SizeofCmsghdr)
	*(*byte)(unsafe.Pointer(dataPtr)) = tos
	_, _, err := conn.WriteMsgUDP(data, oob, addr)
	HandleError(err)
}
