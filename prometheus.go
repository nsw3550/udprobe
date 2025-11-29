package llama

import "github.com/prometheus/client_golang/prometheus"

// Labels we want to include in our metrics. Update if we want to add extra tags / labels.
var llamaLabels = []string{"src_ip", "dst_ip", "src_hostname", "dst_hostname"}

// Packet Loss Percentage
var llamaPacketLoss = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_packet_loss_percentage",
		Help: "Packet loss percentage for a given measurement period.",
	},
	llamaLabels,
)

// Packets Sent
var llamaPacketsSent = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_packets_sent",
		Help: "Number of packets sent for a given measurement period.",
	},
	llamaLabels,
)

// Packets Lost
var llamaPacketsLost = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_packets_lost",
		Help: "Number of packets lost for a given measurement period.",
	},
	llamaLabels,
)

// RTT for packets sent / received
var llamaRTT = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "llama_rtt",
		Help: "RTT for packets sent during a given measurement period.",
	},
	llamaLabels,
)

// Interface for setting metrics. Should make it easier to test.
type MetricSetter interface {
	SetPacketLoss(labels map[string]string, value float64)
	SetPacketsSent(labels map[string]string, value float64)
	SetPacketsLost(labels map[string]string, value float64)
	SetRTT(labels map[string]string, value float64)
}

type PrometheusMetricSetter struct{}

func (p *PrometheusMetricSetter) SetPacketLoss(labels map[string]string, value float64) {
	llamaPacketLoss.With(labels).Set(value)
}

func (p *PrometheusMetricSetter) SetPacketsSent(labels map[string]string, value float64) {
	llamaPacketsSent.With(labels).Set(value)
}

func (p *PrometheusMetricSetter) SetPacketsLost(labels map[string]string, value float64) {
	llamaPacketsLost.With(labels).Set(value)
}

func (p *PrometheusMetricSetter) SetRTT(labels map[string]string, value float64) {
	llamaRTT.With(labels).Set(value)
}

// EmitMetricsFromSummaries updates the Prometheus metrics based on the summaries with the necessary tags
func EmitMetricsFromSummaries(summaries []*Summary, t TagSet, setter MetricSetter) {
	for _, summary := range summaries {
		tags := t[summary.Pd.DstIP.String()]
		labels := prometheus.Labels{
			"src_ip":       summary.Pd.SrcIP.String(),
			"dst_ip":       summary.Pd.DstIP.String(),
			"src_hostname": tags["src_hostname"],
			"dst_hostname": tags["dst_hostname"],
		}
		setter.SetPacketLoss(labels, summary.Loss)
		setter.SetPacketsSent(labels, float64(summary.Sent))
		setter.SetPacketsLost(labels, float64(summary.Lost))
		setter.SetRTT(labels, summary.RTTAvg)
	}
}

func RegisterPrometheus() {
	prometheus.MustRegister(llamaPacketLoss, llamaPacketsSent, llamaPacketsLost, llamaRTT)
}
