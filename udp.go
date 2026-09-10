package udprobe

import (
	"net"

	"golang.org/x/sys/unix" // The successor to syscall
)

// LocalUDPAddr returns the UDPAddr and net for the provided UDPConn.
//
// For UDPConn instances, net is generaly 'udp'.
func LocalUDPAddr(conn *net.UDPConn) (*net.UDPAddr, string, error) {
	addr := conn.LocalAddr()
	network := addr.Network()
	udpAddr, err := net.ResolveUDPAddr(network, addr.String())
	if err != nil {
		return udpAddr, network, err
	}
	return udpAddr, network, nil
}

// SetTos will set the IP_TOS value for the unix socket for the provided conn.
//
// Uses SyscallConn().Control() so the option is set directly on the
// socket's file descriptor. Unlike conn.File(), this does not duplicate the
// descriptor or risk leaving the socket in blocking mode, which would
// disable netpoll and cause read deadlines to be ignored.
func SetTos(conn *net.UDPConn, tos byte) {
	rc, err := conn.SyscallConn()
	HandleError(err)
	var sockErr error
	err = rc.Control(func(fd uintptr) {
		sockErr = unix.SetsockoptByte(int(fd), unix.IPPROTO_IP,
			unix.IP_TOS, tos)
	})
	HandleError(err)
	HandleError(sockErr)
}

// GetTos will get the IP_TOS value for the unix socket for the provided conn.
//
// Uses SyscallConn().Control() for the same reasons as SetTos.
func GetTos(conn *net.UDPConn) byte {
	rc, err := conn.SyscallConn()
	HandleError(err)
	value := 0
	var sockErr error
	err = rc.Control(func(fd uintptr) {
		value, sockErr = unix.GetsockoptInt(int(fd), unix.IPPROTO_IP,
			unix.IP_TOS)
	})
	HandleError(err)
	HandleError(sockErr)
	// Convert it to a byte and return
	return byte(value)
}

// EnableTimestamps enables kernel receive timestamping of packets on the
// provided conn.
//
// The timestamp values can later be extracted in the oob data from
// Receive. Uses SyscallConn().Control() for the same reasons as SetTos.
func EnableTimestamps(conn *net.UDPConn) {
	rc, err := conn.SyscallConn()
	HandleError(err)
	var sockErr error
	err = rc.Control(func(fd uintptr) {
		sockErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET,
			unix.SO_TIMESTAMPNS, 1)
	})
	HandleError(err)
	HandleError(sockErr)
}
