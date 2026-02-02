package udprobe

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	reflectorPacketsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udprobe_reflector_packets_received_total",
		Help: "Total UDP packets received by the reflector.",
	})

	reflectorPacketsReflected = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udprobe_reflector_packets_reflected_total",
		Help: "Packets successfully reflected back to sender.",
	})

	reflectorPacketsBadData = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udprobe_reflector_packets_bad_data_total",
		Help: "Malformed/unparseable packets received.",
	})

	reflectorPacketsThrottled = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udprobe_reflector_packets_throttled_total",
		Help: "Packets dropped due to rate limiting.",
	})

	reflectorTosChanges = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udprobe_reflector_tos_changes_total",
		Help: "ToS bit changes on the socket.",
	})

	reflectorUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "udprobe_reflector_up",
		Help: "Health status: 1 if running, 0 if stopped.",
	})

	reflectorRegisterOnce sync.Once
)

func RegisterReflectorPrometheus() {
	reflectorRegisterOnce.Do(func() {
		prometheus.MustRegister(
			reflectorPacketsReceived,
			reflectorPacketsReflected,
			reflectorPacketsBadData,
			reflectorPacketsThrottled,
			reflectorTosChanges,
			reflectorUp,
		)
	})
}
