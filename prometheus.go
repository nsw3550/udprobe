package udprobe

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Labels we want to include in our metrics. Update if we want to add extra tags / labels.
	udprobeLabels = []string{"src_ip", "dst_ip", "src_hostname", "dst_hostname"}

	// Packet Loss Percentage
	udprobePacketLoss = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "udprobe_packet_loss_percentage",
			Help: "Packet loss percentage for a given measurement period.",
		},
		udprobeLabels,
	)

	// Packets Sent
	udprobePacketsSent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "udprobe_packets_sent",
			Help: "Number of packets sent for a given measurement period.",
		},
		udprobeLabels,
	)

	// Packets Lost
	udprobePacketsLost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "udprobe_packets_lost",
			Help: "Number of packets lost for a given measurement period.",
		},
		udprobeLabels,
	)

	// RTT for packets sent / received
	udprobeRTT = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "udprobe_rtt",
			Help: "RTT for packets sent during a given measurement period.",
		},
		udprobeLabels,
	)

	registerOnce sync.Once
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
	udprobePacketLoss.With(labels).Set(value)
}

func (p *PrometheusMetricSetter) SetPacketsSent(labels map[string]string, value float64) {
	udprobePacketsSent.With(labels).Set(value)
}

func (p *PrometheusMetricSetter) SetPacketsLost(labels map[string]string, value float64) {
	udprobePacketsLost.With(labels).Set(value)
}

func (p *PrometheusMetricSetter) SetRTT(labels map[string]string, value float64) {
	udprobeRTT.With(labels).Set(value)
}

// EmitMetricsFromSummaries updates the Prometheus metrics based on the summaries with the necessary tags
func EmitMetricsFromSummaries(summaries []*Summary, t TagSet, setter MetricSetter) {
	for _, summary := range summaries {
		tags := t.Get(summary.Pd.DstIP.String())
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
	registerOnce.Do(func() {
		prometheus.MustRegister(udprobePacketLoss, udprobePacketsSent, udprobePacketsLost, udprobeRTT)
	})
}
