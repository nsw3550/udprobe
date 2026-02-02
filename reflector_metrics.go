package llama

import "github.com/prometheus/client_golang/prometheus"

var reflectorPacketsReceived = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "llama_reflector_packets_received_total",
	Help: "Total UDP packets received by the reflector.",
})

var reflectorPacketsReflected = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "llama_reflector_packets_reflected_total",
	Help: "Packets successfully reflected back to sender.",
})

var reflectorPacketsBadData = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "llama_reflector_packets_bad_data_total",
	Help: "Malformed/unparseable packets received.",
})

var reflectorPacketsThrottled = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "llama_reflector_packets_throttled_total",
	Help: "Packets dropped due to rate limiting.",
})

var reflectorTosChanges = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "llama_reflector_tos_changes_total",
	Help: "ToS bit changes on the socket.",
})

var reflectorUp = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "llama_reflector_up",
	Help: "Health status: 1 if running, 0 if stopped.",
})

func RegisterReflectorPrometheus() {
	prometheus.MustRegister(
		reflectorPacketsReceived,
		reflectorPacketsReflected,
		reflectorPacketsBadData,
		reflectorPacketsThrottled,
		reflectorTosChanges,
		reflectorUp,
	)
}
