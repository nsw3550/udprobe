package udprobe

import (
	"testing"
)

var exampleTargetConfig = TargetConfig{
	IP:   "1.2.3.4",
	Port: 1234,
	Tags: Tags{"tag1": "value1", "tag2": "value2"},
}
var exampleTargetConfig2 = TargetConfig{
	IP:   "127.0.0.1",
	Port: 32000,
	Tags: Tags{"cat": "land", "bird": "air"},
}
var exampleTargetConfig3 = TargetConfig{
	IP:   "192.168.0.1",
	Port: 0,
	Tags: Tags{"foo": "bar", "baz": "qux"},
}

var exampleTargetSet = TargetSet{exampleTargetConfig, exampleTargetConfig2}
var exampleTargetSet2 = TargetSet{exampleTargetConfig3}

var exampleTargetsConfig = TargetsConfig{
	"example": exampleTargetSet,
	"moar":    exampleTargetSet2,
}

var exampleLegacyConfig = `
1.2.3.4:
    my_tag: my_value
    foo:    bar
127.0.0.1:
    tag1:   value1
    baz:    qux
`

func TestTargetConfigAddrString(t *testing.T) {
	expected := "1.2.3.4:1234"
	result := exampleTargetConfig.AddrString()
	if result != expected {
		t.Error("Target addr not formatted correctly. Expected",
			expected, "got", result)
	}
	// Make sure it returns non-nil for zero values
	tc := TargetConfig{}
	if tc.AddrString() == "" {
		t.Error("Zero values returned nil instead of string value. Got:", tc.AddrString())
	}
}

func TestTargetConfigResolveUDPAddr(t *testing.T) {
	_, err := exampleTargetConfig.ResolveUDPAddr()
	if err != nil {
		t.Error("Target couldn't be converted to UDPAddr:", err)
	}
	// Make sure it doesn't fail for zero values (:0)
	tc := TargetConfig{}
	_, err = tc.ResolveUDPAddr()
	if err != nil {
		t.Error("Zero value couldn't be converted to UDPAddr:", err)
	}
}

func TestTargetSetTagSet(t *testing.T) {
	tagset := exampleTargetSet.TagSet("")
	_, ok := tagset["1.2.3.4"]
	if !ok {
		t.Error("Parsed value was not populated")
	}
	// TODO(nwinemiller): Add some deeper testing here
}

func TestTargetSetIntoTagSet(t *testing.T) {
	tagset := make(TagSet)
	tagset["example"] = Tags{"mytag": "myvalue"}
	exampleTargetSet.IntoTagSet(tagset, "")
	_, ok := tagset["example"]
	if !ok {
		t.Error("Prepopulated value was erased")
	}
	_, ok = tagset["1.2.3.4"]
	if !ok {
		t.Error("Parsed value was not populated")
	}
}

func TestTargetSetListTarget(t *testing.T) {
	expected := []string{"1.2.3.4:1234", "127.0.0.1:32000"}
	result := exampleTargetSet.ListTargets()
	// TODO(nwinemiller): Do deeper evaluation here.
	if len(expected) != len(result) {
		t.Error("Expected:", expected, "but got:", result)
	}
}

func TestTargetSetListResolvedTargets(t *testing.T) {
	addrs, err := exampleTargetSet.ListResolvedTargets()
	if err != nil {
		t.Error(err)
	}
	if len(addrs) != 2 {
		t.Error("Expected 2 UDPAddrs but got", len(addrs))
	}
}

func TestTargetsConfig(t *testing.T) {
	tagset := exampleTargetsConfig.TagSet("")
	_, ok := tagset["1.2.3.4"]
	if !ok {
		t.Error("Parsed value was not populated")
	}
}

func TestTargetsConfigIntoTagSet(t *testing.T) {
	tagset := make(TagSet)
	tagset["example"] = Tags{"mytag": "myvalue"}
	exampleTargetsConfig.IntoTagSet(tagset, "")
	_, ok := tagset["example"]
	if !ok {
		t.Error("Prepopulated value was erased")
	}
	_, ok = tagset["1.2.3.4"]
	if !ok {
		t.Error("Parsed value was not populated")
	}
}

func TestNewDefaultCollectorConfig(t *testing.T) {
	// Just make sure it comes back with no errors.
	// This also tests `NewCollectorConfig
	_, err := NewDefaultCollectorConfig()
	if err != nil {
		t.Error(err)
	}
}

func TestNewLegacyCollectorConfig(t *testing.T) {
	// Just make sure it returns with no errors.
	_, err := NewLegacyCollectorConfig([]byte(exampleLegacyConfig))
	if err != nil {
		t.Error(err)
	}
}

func TestLegacyCollectorConfigToDefaultCollectorConfig(t *testing.T) {
	lcc, _ := NewLegacyCollectorConfig([]byte(exampleLegacyConfig))
	// Just make sure it converts
	_, err := lcc.ToDefaultCollectorConfig(1234)
	if err != nil {
		t.Error(err)
	}
}

func TestTargetSetTagSetWithGlobalSrcHostname(t *testing.T) {
	ts := TargetSet{
		{IP: "1.2.3.4", Port: 1234, Tags: Tags{"dst_hostname": "server-1"}},
	}
	tagset := ts.TagSet("collector-1")

	if tagset["1.2.3.4"]["src_hostname"] != "collector-1" {
		t.Error("Expected src_hostname to be 'collector-1', got:", tagset["1.2.3.4"]["src_hostname"])
	}
}

func TestTargetSetTagSetSrcHostnameOverride(t *testing.T) {
	ts := TargetSet{
		{IP: "1.2.3.4", Port: 1234, Tags: Tags{
			"dst_hostname": "server-1",
			"src_hostname": "specific-collector",
		}},
	}
	tagset := ts.TagSet("global-collector")

	if tagset["1.2.3.4"]["src_hostname"] != "specific-collector" {
		t.Error("Expected src_hostname to be 'specific-collector', got:", tagset["1.2.3.4"]["src_hostname"])
	}
}

func TestTargetSetTagSetEmptyGlobalSrcHostname(t *testing.T) {
	ts := TargetSet{
		{IP: "1.2.3.4", Port: 1234, Tags: Tags{"dst_hostname": "server-1"}},
	}
	tagset := ts.TagSet("")

	if tagset["1.2.3.4"]["src_hostname"] != "" {
		t.Error("Expected src_hostname to be empty, got:", tagset["1.2.3.4"]["src_hostname"])
	}
}

func TestTargetSetTagSetMixedSrcHostname(t *testing.T) {
	ts := TargetSet{
		{IP: "1.2.3.4", Port: 1234, Tags: Tags{"dst_hostname": "server-1"}},
		{IP: "1.2.3.5", Port: 1234, Tags: Tags{
			"dst_hostname": "server-2",
			"src_hostname": "specific",
		}},
	}
	tagset := ts.TagSet("global-collector")

	if tagset["1.2.3.4"]["src_hostname"] != "global-collector" {
		t.Error("Expected src_hostname to be 'global-collector' for 1.2.3.4, got:", tagset["1.2.3.4"]["src_hostname"])
	}
	if tagset["1.2.3.5"]["src_hostname"] != "specific" {
		t.Error("Expected src_hostname to be 'specific' for 1.2.3.5, got:", tagset["1.2.3.5"]["src_hostname"])
	}
}
