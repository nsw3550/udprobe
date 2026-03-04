package udprobe

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigFromDefault(t *testing.T) {
	c := &Collector{}
	err := c.loadConfigFromDefault()
	if err != nil {
		t.Errorf("loadConfigFromDefault failed: %v", err)
	}
	if c.cfg == nil {
		t.Error("loadConfigFromDefault did not set c.cfg")
	}
}

func TestLoadConfigFromData(t *testing.T) {
	c := &Collector{}
	yamlData := `
targets:
  test:
    - ip: 127.0.0.1
      port: 8100
tests:
  - targets: test
    port_group: default
    rate_limit: default
port_groups:
  default:
    - port: default
      count: 1
ports:
  default:
    ip: 0.0.0.0
    port: 0
    tos: 0
    timeout: 1000
rate_limits:
  default:
    cps: 10
summarization:
  interval: 10
  handlers: 1
`
	err := c.loadConfigFromData([]byte(yamlData))
	if err != nil {
		t.Errorf("loadConfigFromData failed: %v", err)
	}
	if c.cfg == nil {
		t.Error("loadConfigFromData did not set c.cfg")
	}

	// Test legacy config
	legacyData := `
127.0.0.1:
  tag1: value1
`
	err = c.loadConfigFromData([]byte(legacyData))
	if err != nil {
		t.Errorf("loadConfigFromData legacy failed: %v", err)
	}
	if c.cfg == nil {
		t.Error("loadConfigFromData legacy did not set c.cfg")
	}

	// Test invalid config
	err = c.loadConfigFromData([]byte("[invalid yaml"))
	if err == nil {
		t.Error("loadConfigFromData should have failed for invalid yaml")
	}
}

func TestLoadConfigFromPath(t *testing.T) {
	c := &Collector{}
	tmpFile, err := os.CreateTemp("", "udprobe-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	yamlData := `
targets:
  test:
    - ip: 127.0.0.1
      port: 8100
`
	if _, err := tmpFile.Write([]byte(yamlData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	err = c.loadConfigFromPath(tmpFile.Name())
	if err != nil {
		t.Errorf("loadConfigFromPath failed: %v", err)
	}
	if c.cfg == nil {
		t.Error("loadConfigFromPath did not set c.cfg")
	}

	// Test non-existent file
	err = c.loadConfigFromPath("non-existent-file")
	if err == nil {
		t.Error("loadConfigFromPath should have failed for non-existent file")
	}
}

func TestSetupTagSet(t *testing.T) {
	c := &Collector{}
	yamlData := `
targets:
  test:
    - ip: 127.0.0.1
      port: 8100
      tags:
        foo: bar
`
	_ = c.loadConfigFromData([]byte(yamlData))
	c.SetupTagSet()
	if c.ts == nil {
		t.Fatal("SetupTagSet did not set c.ts")
	}
	if c.ts["127.0.0.1"]["foo"] != "bar" {
		t.Errorf("Expected tag foo=bar, got %v", c.ts["127.0.0.1"]["foo"])
	}
}

func TestSetupSummarizer(t *testing.T) {
	c := &Collector{}
	yamlData := `
summarization:
  interval: 10
  handlers: 2
`
	_ = c.loadConfigFromData([]byte(yamlData))
	c.SetupSummarizer()
	if c.s == nil {
		t.Error("SetupSummarizer did not set c.s")
	}
	if len(c.rh) != 2 {
		t.Errorf("Expected 2 result handlers, got %d", len(c.rh))
	}
}

func TestSetupAPI(t *testing.T) {
	c := &Collector{}
	yamlData := `
api:
  bind: ":8080"
summarization:
  interval: 10
  handlers: 1
`
	_ = c.loadConfigFromData([]byte(yamlData))
	c.SetupAPI()
	if c.api == nil {
		t.Error("SetupAPI did not set c.api")
	}
	if c.s == nil {
		t.Error("SetupAPI should have called SetupSummarizer and set c.s")
	}
}

func TestCreateRateLimiter(t *testing.T) {
	c := &Collector{}
	yamlData := `
rate_limits:
  test:
    cps: 50
`
	_ = c.loadConfigFromData([]byte(yamlData))
	rl := c.createRateLimiter("test")
	if rl == nil {
		t.Fatal("createRateLimiter returned nil")
	}
	if rl.Limit() != 50 {
		t.Errorf("Expected limit 50, got %v", rl.Limit())
	}
}

func TestCreatePortOnRunner(t *testing.T) {
	c := &Collector{}
	runner := NewTestRunner(nil, nil)
	p := PortConfig{
		IP:      "127.0.0.1",
		Port:    0,
		Tos:     0,
		Timeout: 500,
	}
	c.createPortOnRunner(runner, p)
	// We can't easily inspect runner's internal ports as they are not exported,
	// but we can at least ensure it doesn't crash.
}

func TestCreatePortGroupOnRunner(t *testing.T) {
	c := &Collector{}
	yamlData := `
port_groups:
  pg1:
    - port: p1
      count: 2
ports:
  p1:
    ip: 0.0.0.0
    port: 0
    tos: 0
    timeout: 1000
`
	_ = c.loadConfigFromData([]byte(yamlData))
	runner := NewTestRunner(nil, nil)
	c.createPortGroupOnRunner(runner, "pg1")
}

func TestSetupTestRunner(t *testing.T) {
	c := &Collector{}
	yamlData := `
targets:
  t1:
    - ip: 127.0.0.1
      port: 8100
port_groups:
  pg1:
    - port: p1
      count: 1
ports:
  p1:
    ip: 0.0.0.0
    port: 0
    tos: 0
    timeout: 1000
rate_limits:
  rl1:
    cps: 10
`
	_ = c.loadConfigFromData([]byte(yamlData))
	testCfg := TestConfig{
		Targets:   "t1",
		PortGroup: "pg1",
		RateLimit: "rl1",
	}
	c.cbc = make(chan *InFlightProbe, 10)
	c.SetupTestRunner(testCfg)
	if len(c.runners) != 1 {
		t.Errorf("Expected 1 runner, got %d", len(c.runners))
	}
}

func TestSetupRunStop(t *testing.T) {
	c := &Collector{}
	// We need to bypass LoadConfig's reliance on flags for this test to be robust
	yamlData := `
targets:
  t1:
    - ip: 127.0.0.1
      port: 8100
tests:
  - targets: t1
    port_group: pg1
    rate_limit: rl1
port_groups:
  pg1:
    - port: p1
      count: 1
ports:
  p1:
    ip: 127.0.0.1
    port: 0
    tos: 0
    timeout: 1000
rate_limits:
  rl1:
    cps: 10
summarization:
  interval: 10
  handlers: 1
api:
  bind: "127.0.0.1:0"
`
	_ = c.loadConfigFromData([]byte(yamlData))

	// Manually perform setup to avoid LoadConfig which uses flags
	c.SetupTagSet()
	c.SetupTestRunners()
	c.SetupSummarizer()
	c.SetupAPI()

	c.Run()
	// Let it run for a tiny bit
	time.Sleep(10 * time.Millisecond)
	c.Stop()
}

func TestReload(t *testing.T) {
	c := &Collector{}
	yamlData := `
targets:
  t1:
    - ip: 127.0.0.1
      port: 8100
tests:
  - targets: t1
    port_group: pg1
    rate_limit: rl1
port_groups:
  pg1:
    - port: p1
      count: 1
ports:
  p1:
    ip: 127.0.0.1
    port: 0
    tos: 0
    timeout: 1000
rate_limits:
  rl1:
    cps: 10
summarization:
  interval: 10
  handlers: 1
api:
  bind: "127.0.0.1:0"
`
	// Initial setup
	_ = c.loadConfigFromData([]byte(yamlData))
	c.SetupTagSet()
	c.SetupTestRunners()
	c.SetupSummarizer()
	c.SetupAPI()

	// Reload. We need to mock or ensure LoadConfig doesn't crash.
	// Since we can't easily mock it without refactoring, we'll ensure c.cfg is still valid.
	// But Reload calls LoadConfig which might overwrite c.cfg.
	// If no flags are set, it loads default.
	c.Reload()
}

func TestLoadConfigWithFlag(t *testing.T) {
	c := &Collector{}
	oldConfigFile := *configFile
	defer func() { *configFile = oldConfigFile }()

	tmpFile, _ := os.CreateTemp("", "udprobe-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("targets: {}"))
	tmpFile.Close()

	*configFile = tmpFile.Name()
	c.LoadConfig()
	if c.cfg == nil {
		t.Error("LoadConfig with flag failed to set c.cfg")
	}
}

func TestSetup(t *testing.T) {
	c := &Collector{}
	// Setup calls LoadConfig. If no flag is set, it loads default config.
	// We'll ensure it doesn't crash.
	c.Setup()
}

func TestLoadConfigFromFlag(t *testing.T) {
	c := &Collector{}
	// configFile is a pointer to a string flag
	oldConfigFile := *configFile
	defer func() { *configFile = oldConfigFile }()

	// Test when flag is empty
	*configFile = ""
	err := c.loadConfigFromFlag()
	if err == nil {
		t.Error("loadConfigFromFlag should have failed when flag is empty")
	}

	// Test when flag is set
	tmpFile, _ := os.CreateTemp("", "udprobe-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("targets: {}"))
	tmpFile.Close()

	*configFile = tmpFile.Name()
	err = c.loadConfigFromFlag()
	if err != nil {
		t.Errorf("loadConfigFromFlag failed: %v", err)
	}
}
