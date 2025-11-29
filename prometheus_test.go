package llama

import (
	"net"
	"testing"
)

type MockMetricSetter struct {
	CalledWith []struct {
		Metric string
		Labels map[string]string
		Value  float64
	}
}

func (m *MockMetricSetter) SetPacketLoss(labels map[string]string, value float64) {
	m.CalledWith = append(m.CalledWith, struct {
		Metric string
		Labels map[string]string
		Value  float64
	}{"PacketLoss", labels, value})
}
func (m *MockMetricSetter) SetPacketsLost(labels map[string]string, value float64) {
	m.CalledWith = append(m.CalledWith, struct {
		Metric string
		Labels map[string]string
		Value  float64
	}{"PacketsLost", labels, value})
}
func (m *MockMetricSetter) SetPacketsSent(labels map[string]string, value float64) {
	m.CalledWith = append(m.CalledWith, struct {
		Metric string
		Labels map[string]string
		Value  float64
	}{"PacketsSent", labels, value})
}
func (m *MockMetricSetter) SetRTT(labels map[string]string, value float64) {
	m.CalledWith = append(m.CalledWith, struct {
		Metric string
		Labels map[string]string
		Value  float64
	}{"RTT", labels, value})
}

func TestEmitMetricsFromSummary(t *testing.T) {
	m := &MockMetricSetter{}

	mockPD := &PathDist{
		SrcIP: net.ParseIP("1.1.1.1"),
		DstIP: net.ParseIP("2.2.2.2"),
	}

	mockSummary := &Summary{
		Pd:     mockPD,
		Loss:   10.0,
		Sent:   100,
		Lost:   10,
		RTTAvg: 10.5,
	}

	mockTagSet := TagSet{
		"2.2.2.2": { // Keyed by DstIP
			"src_hostname": "test-source",
			"dst_hostname": "test-destination",
		},
	}

	EmitMetricsFromSummaries([]*Summary{mockSummary}, mockTagSet, m)

	expectedLabels := map[string]string{
		"src_ip":       "1.1.1.1",
		"dst_ip":       "2.2.2.2",
		"src_hostname": "test-source",
		"dst_hostname": "test-destination",
	}

	expected := []struct {
		Metric string
		Value  float64
	}{
		{"PacketLoss", 10.0},
		{"PacketsSent", float64(100)},
		{"PacketsLost", float64(10)},
		{"RTT", 10.5},
	}

	for i, expectedCall := range expected {
		actualCall := m.CalledWith[i]

		// Assert 2a: Metric Name
		if actualCall.Metric != expectedCall.Metric {
			t.Errorf("Call %d: Expected metric %s, got %s", i, expectedCall.Metric, actualCall.Metric)
		}

		// Assert 2b: Value (using a tolerance for float comparison)
		if actualCall.Value != expectedCall.Value { // For simplicity, direct compare works here, but use tolerance for real-world floats
			t.Errorf("Call %d (%s): Expected value %f, got %f", i, expectedCall.Metric, expectedCall.Value, actualCall.Value)
		}

		// Assert 2c: Labels
		// Check if the map content is deeply equal to the expected labels
		if !mapsEqual(actualCall.Labels, expectedLabels) {
			t.Errorf("Call %d (%s): Labels mismatch. Expected %v, Got %v", i, expectedCall.Metric, expectedLabels, actualCall.Labels)
		}
	}
}

// mapsEqual is a helper function to compare string maps for deep equality
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
