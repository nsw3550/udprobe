// Functionality for sending and receiving UDP probes on a socket.
package udprobe

import (
	"context"
	"errors"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
	pb "github.com/nsw3550/udprobe/proto"
	"google.golang.org/protobuf/proto"
)

// Port represents a socket and its associated caching, inputs, and outputs.
type Port struct {
	tosend      chan *net.UDPAddr // A channel for receiving targets
	conn        *net.UDPConn      // The socket on which to send/receive
	cache       *ttlcache.Cache[string, *InFlightProbe]
	stop        chan bool           // A signal to stop processing
	cbc         chan *InFlightProbe // Callback channel for sending expired Probes
	readTimeout time.Duration       // How long to wait for reads
	basePD      *PathDist           // A partially filled PathDist based on conn
}

// srcPD creates a PathDist based on the known socket details for the port.
func (p *Port) srcPD() *PathDist {
	if p.basePD != nil {
		// Just return the saved base, so we don't waste time
		return p.basePD
	}
	udpAddr, network, err := LocalUDPAddr(p.conn)
	HandleError(err)
	pd := PathDist{
		SrcIP:   udpAddr.IP,
		SrcPort: udpAddr.Port,
		Proto:   network,
	}
	p.basePD = &pd // Save for future use
	return &pd
}

// pd will provide a completed PathDist based on the associated p.conn and
// the provided net.UDPAddr.
func (p *Port) pd(dst *net.UDPAddr) *PathDist {
	// TODO(nwinemiller): Since many of these are going to be repeats, keep these
	//      around for reuse. That also allows memory pointers to be used
	//      for equality/summarization later.
	pathDist := &PathDist{
		SrcIP:   p.srcPD().SrcIP,
		SrcPort: p.srcPD().SrcPort,
		Proto:   p.srcPD().Proto,
		DstIP:   dst.IP,
		DstPort: dst.Port,
	}
	return pathDist
}

// ToS provides the currently active ToS byte value for the port's conn.
func (p *Port) Tos() byte {
	val := GetTos(p.conn)
	return val
}

// Send waits to get UDPAddr targets and sends probes to them using the
// associated Port.
//
// After sending the probe, it is added to a cache with a unique ID, which is
// used for retrieving later. The cache will also utilize a timeout to expire
// probes that haven't returned in time.
func (p *Port) Send() {
	go p.send()
}

func (p *Port) send() {
	for {
		select {
		case <-p.stop:
			LogInfo("Stopping Port.send for " + p.conn.LocalAddr().String())
			return // Discontinue sending
		case addr := <-p.tosend:
			if addr.IP == nil {
				LogWarning("Skipping target with nil IP: " + addr.String())
				continue
			}
			pd := p.pd(addr)
			tos := p.Tos()
			key := NewID()
			// NOTE: The more time spent before sending, the more stale
			//       this will get. Not critical, but a consideration.
			now := NowUint64()
			probe := InFlightProbe{
				Pd:    pd,
				CSent: now,
				Tos:   tos,
			}
			// Add the probe to cache
			// TODO(nwinemiller): Might want to make this async in the future to avoid
			//             making `now` more stale as things are going on.
			p.cache.Set(key, &probe, ttlcache.DefaultTTL)
			signature := IDToBytes(key)
			var padding [1000]byte
			data := &pb.Probe{
				Signature: signature[:],
				Tos:       uint32(tos),
				Sent:      now,
				// TODO(nwinemiller): This should be customizable, and relative to
				//			   to the rest of the probe. This should really
				//             be used to fill to a maximum size.
				//			   Likely based on the return from Marshal.
				Padding: padding[:],
			}
			packedData, err := proto.Marshal(data)
			HandleError(err)
			// Send the probe
			_, err = p.conn.WriteToUDP(packedData, addr)
			if err != nil {
				// A closed conn normally means stop was signaled (the stop
				// watcher closes the conn on stop). Exit cleanly rather than
				// treating it as a fatal error. Any other error is fatal.
				if netErr, ok := err.(net.Error); ok && strings.Contains(
					netErr.Error(), "use of closed network connection") {
					LogInfo("Conn closed while sending on: " +
						p.conn.LocalAddr().String())
					return
				}
				HandleError(err)
			}
			// TODO(nwinemiller): Log rate of `packets_sent`
		}
	}
}

// Recv listens on the Port for returning probes and updates them in the cache.
//
// Once probes are received, they are located in the cache, updated, and then
// set for immediate expiration. If a probe is received but has no entry in
// the cache, it most likely exceeded the timeout.
func (p *Port) Recv() {
	go p.recv()
}

func (p *Port) recv() {
	dataBuf := make([]byte, 4096) // Reuse this for the received data
	// This will be implemented for timestamps in the future
	oobBuf := make([]byte, 4096) // Reuse this for the received oob data
	// stopRecv performs any cleanup needed when receiving stops, so that
	// both the stop-signal path and the closed-conn path behave the same.
	stopRecv := func() {
		// Don't process expirations anymore
		// This prevents outstanding probes from reporting as loss
		p.cache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason,
			item *ttlcache.Item[string, *InFlightProbe]) {
		})
	}
	for {
		select {
		case <-p.stop:
			LogInfo("Stopping Port.recv for: " + p.conn.LocalAddr().String())
			stopRecv()
			return // Stop receiving
		default:
			// This is a specific point in time, so it needs to be refreshed
			timeout := time.Now().Add(p.readTimeout)
			err := p.conn.SetReadDeadline(timeout)
			HandleError(err)
			// TODO(nwinemiller):
			// This is very similar to `reflector.Receive` except for timeout
			// handling. Should consolidate these at some point in UDP.
			// Ignoring `oobLen` and `flags`for now
			// We don't need `addr since we're matching on the signature
			// NOTE(nwinemiller): Previously, on stop, a process would
			//   occasionally get stuck here, on the underlying Recvmsg
			//   call in syscall, ignoring the read deadline. The root
			//   cause was the socket being left in blocking mode by
			//   conn.File()-based socket option helpers, which disables
			//   netpoll and deadlines. That is fixed by using
			//   SyscallConn().Control() in udp.go, and stopWatch now
			//   closes the conn on stop to force-unblock any wedged read.
			dataLen, _, _, _, err := p.conn.ReadMsgUDP(dataBuf, oobBuf)
			if err != nil {
				// Check if it's a networking error
				netErr, ok := err.(net.Error)
				if ok && netErr.Timeout() {
					// It's a timeout, so we've waited long enough, restart the loop
					continue
				} else if ok && strings.Contains(netErr.Error(),
					"use of closed network connection") {
					// The connection was closed, which normally means stop was
					// signaled (the stop watcher closes the conn on stop).
					// Since stop is the only expected reason for this, exit
					// cleanly rather than treating it as a fatal error.
					LogInfo("Conn closed while receiving on: " +
						p.conn.LocalAddr().String())
					stopRecv()
					return
				} else {
					// Some other problem
					HandleFatalErrorMsg(err, "Failure while listening on "+p.conn.LocalAddr().String())
				}
			}
			data := dataBuf[0:dataLen]
			udpData := &pb.Probe{}
			err = proto.Unmarshal(data, udpData)
			HandleMinorErrorMsg(err, "failed to unmarshal probe data")
			id := string(udpData.Signature[:])
			// TODO(nwinemiller): Should be doing something about this error
			item := p.cache.Get(id)
			if item == nil || item.IsExpired() {
				// This means it expired already or doesn't exist
				// so there's nothing to do.
				// TODO(nwinemiller): Log/stat on occurrences of this
				continue
			}
			// TODO(nwinemiller): Make wish to make a `ProbeCache` that does this
			//             automatically under the hood.
			probe, err := IfaceToInFlightProbe(item.Value())
			HandleMinorErrorMsg(err, "failed to convert interface to InFlightProbe")
			probe.CRcvd = NowUint64()
			probe.ReflectorRcvd = udpData.Rcvd
			// Error would be if the key didn't exist, meaning it expired
			// since the Get above. Rare but possible. Acceptable for now.
			// TODO(nwinemiller): Log/stat on occurrences of this
			p.cache.Set(id, probe, ExpireNow)
			// TODO(nwinemiller): Log rate of `packets_received`
		}
	}
}

// done receives entries in the cache that have expired and passes them to
// the Port's cbc (callback channel)
//
// This basically just exists to the do the type conversion and pass to the
// channel.
func (p *Port) done(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, *InFlightProbe]) {
	p.cbc <- item.Value()
}

// InFlightProbe represents a single UDP probe that was sent from, and (hopefully)
// received back, a Port.
type InFlightProbe struct {
	Pd            *PathDist
	CSent         uint64
	CRcvd         uint64
	ReflectorRcvd uint64
	Tos           byte
}

// PathDist -> Path Distinguisher, uniquely IDs the components that determine
// path selection.
type PathDist struct {
	SrcIP   net.IP
	SrcPort int
	DstIP   net.IP
	DstPort int
	Proto   string // 'udp' generally
}

// Cleanup will close the connection and release the cache.
// This would be triggered as a result of garbage collection, and would likely
// be better suited elsewhere. However, this seems like a fairly simple option
// for now, to avoid needing locks and conflicts between send/recv.
//
// The connection may already be closed (by stopWatch on stop); that case is
// handled gracefully since a double-close just returns an error handled here.
func cleanup(port *Port) {
	LogInfo("Started closing port on: " + port.conn.LocalAddr().String())
	err := port.conn.Close()
	HandleMinorErrorMsg(err, "failed to close port")
	// This might not actually be necessary, if we've already stopped
	// using this whole thing. But doesn't hurt either.
	port.cache = nil // Dereference the cache
	LogInfo("Finished closing port on: " + port.conn.LocalAddr().String())
}

// stopWatch waits for the stop signal and then closes the Port's connection.
//
// Closing the connection force-unblocks any in-progress ReadMsgUDP or
// WriteToUDP call, even in the rare case where the read deadline has been
// silently disabled (e.g., the fd being left in blocking mode). This makes
// stopping deterministic, rather than relying solely on read deadlines.
// It is started by NewPort and is intended to run for the lifetime of the
// Port.
func (p *Port) stopWatch() {
	<-p.stop
	LogInfo("Closing conn for stop on: " + p.conn.LocalAddr().String())
	err := p.conn.Close()
	HandleMinorErrorMsg(err, "failed to close conn during stop")
}

// New creates and returns a new Port with associated inputs, outputs,
// and caching mechanisms.
func NewPort(conn *net.UDPConn, tosend chan *net.UDPAddr, stop chan bool,
	cbc chan *InFlightProbe, cTimeout time.Duration, cCleanRate time.Duration,
	readTimeout time.Duration,
) *Port {
	// Create the cache
	cache := ttlcache.New[string, *InFlightProbe](
		ttlcache.WithTTL[string, *InFlightProbe](cTimeout),
	)
	// Create the port
	port := Port{
		tosend: tosend, conn: conn, cache: cache,
		stop: stop, cbc: cbc, readTimeout: readTimeout,
	}
	// Used for wrapping the callback channel
	port.cache.OnEviction(port.done)
	go cache.Start()
	// Close the conn on stop, so that a wedged read/write is unblocked
	// deterministically.
	go port.stopWatch()
	// Ensure that when the port is stopped, we cleanup.
	// This happens on GC, so it may be delayed for a bit.
	runtime.SetFinalizer(&port, cleanup)
	return &port
}

// NewDefault creates a new Port using default settings.
func NewDefault(tosend chan *net.UDPAddr, stop chan bool,
	cbc chan *InFlightProbe,
) *Port {
	// Create a default UDPConn
	udpAddr, err := net.ResolveUDPAddr("udp", DefaultAddrStr)
	HandleError(err)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	HandleError(err)
	// These two are unnecessary, but being explicit
	err = udpConn.SetReadBuffer(DefaultRcvBuff)
	HandleError(err)
	SetTos(udpConn, DefaultTos)
	// TODO(nwinemiller): Update to allow no args, and setting later if desired.
	port := NewPort(
		udpConn,
		tosend,
		stop,
		cbc,
		DefaultCacheTimeout,
		DefaultCacheCleanRate,
		DefaultReadTimeout,
	)
	return port
}

// IfaceToInFlightProbe attempts to convert an anonymous object to a InFlightProbe, and returns
// and error if the operation failed.
func IfaceToInFlightProbe(iface interface{}) (*InFlightProbe, error) {
	probe, ok := iface.(*InFlightProbe)
	if ok {
		return probe, nil
	} else {
		return probe, errors.New("object provided is not a InFlightProbe")
	}
}
