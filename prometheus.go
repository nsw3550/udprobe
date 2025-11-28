package llama

import "github.com/prometheus/client_golang/prometheus"

var llamaLabels = []string{"src_ip", "dst_ip", "src_hostname", "dst_hostname"}

var llamaPacketLoss = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_packet_loss_percentage",
		Help: "Packet loss percentage for a given measurement period.",
	},
	llamaLabels,
)

var llamaPacketsSent = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_packets_sent",
		Help: "Number of packets sent for a given measurement period.",
	},
	llamaLabels,
)

var llamaPacketsLost = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_packets_lost",
		Help: "Number of packets lost for a given measurement period.",
	},
	llamaLabels,
)

var llamaRTT = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_rtt",
		Help: "RTT for packets sent during a given measurement period.",
	},
	llamaLabels,
)

// EmitMetricsFromSummaries updates the Prometheus metrics based on the summaries with the necessary tags
func EmitMetricsFromSummaries(summaries []*Summary, t TagSet) {
	for _, summary := range summaries {
		tags := t[summary.Pd.DstIP.String()]
		labels := prometheus.Labels{
			"src_ip":       summary.Pd.SrcIP.String(),
			"dst_ip":       summary.Pd.DstIP.String(),
			"src_hostname": tags["src_hostname"],
			"dst_hostname": tags["dst_hostname"],
		}

		llamaPacketLoss.With(labels).Set(summary.Loss)
		llamaPacketsSent.With(labels).Set(float64(summary.Sent))
		llamaPacketsLost.With(labels).Set(float64(summary.Lost))
		llamaRTT.With(labels).Set(summary.RTTAvg)
	}
}

func RegisterPrometheus() {
	prometheus.MustRegister(llamaPacketLoss, llamaPacketsSent, llamaPacketsLost, llamaRTT)
}
