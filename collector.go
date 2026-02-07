// UDProbe Collector sends UDP probes to a set of target reflectors and provides
// statistics about their latency and reachability via an API.
package udprobe

import (
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"golang.org/x/time/rate"
)

const DEFAULT_CHANNEL_SIZE int64 = 100 // Default size used for buffered channels.

// TODO(nwinemiller): This really shouldn't be in here, and should be provided from the
//
//	cmd tools or higher up.
//
// Flags
var configFile = flag.String("udprobe.config", "", "Config file to load from")

// Temporary until this is provided in the config
var dstPort = flag.Int64("udprobe.dst-port", 8100, "Port to send probes to. Only applies if legacy config provided.")

// Collector reads a YAML configuration, performs UDP probe tests against
// targets, and provides summaries of the results via a JSON HTTP API.
type Collector struct {
	cfg *CollectorConfig
	ts  TagSet
	api *API
	// TODO(nwinemiller): Might want these to be named, for clarity in logging
	//      and doing any restarting.
	runners []*TestRunner
	// TODO(nwinemiller): Keeping cbc around here feels dirty and unneeded, as it's
	//      only temporarily needed during setup. But it does the trick for
	//      now. Perhaps find a cleaner way in the future.
	cbc chan *Probe
	s   *Summarizer
	rh  []*ResultHandler
}

// LoadConfig loads the collector's configuration from CLI flag if provided,
// otherwise the default.
func (c *Collector) LoadConfig() {
	LogInfo("Loading collector config")
	// Try loading from flag first
	if *configFile != "" {
		err := c.loadConfigFromFlag()
		if err == nil {
			return
		}
		HandleFatalErrorMsg(err, "Failed to load configuration")
		// If that wasn't provided, load the default
	} else {
		LogInfo("No udprobe.config provided; loading default config")
		err := c.loadConfigFromDefault()
		if err == nil {
			return
		}
		HandleFatalErrorMsg(err, "Failed to load configuration")
	}
}

// loadConfigFromFlag attempts to parse and load the configuration file
// provided by the `udprobe.config` CLI flag, returning an error if unsuccessful.
func (c *Collector) loadConfigFromFlag() error {
	if *configFile != "" {
		return c.loadConfigFromPath(*configFile)
	} else {
		return fmt.Errorf("config file not provided via flag")
	}
}

// loadConfigFromPath attempts to parse and load a configuration file from
// the provided string path, returning an error is unsuccessful.
func (c *Collector) loadConfigFromPath(path string) error {
	// Read in the config
	data, err := ioutil.ReadFile(path)
	// If we can't read it, bubble that up
	if err != nil {
		return err
	}
	// Otherwise, try loading the config
	return c.loadConfigFromData(data)
}

// loadConfigFromDefault simply loads the default configuration, returning an
// error if unsuccessful for some reason.
func (c *Collector) loadConfigFromDefault() error {
	cfg, err := NewDefaultCollectorConfig()
	if err != nil {
		return err
	}
	c.cfg = cfg
	return nil
}

// loadConfigFromData attempts to parse and load a configuration from data
// that is already in byte slice form, returning an error is unsuccessful.
func (c *Collector) loadConfigFromData(data []byte) error {
	// Try parsing as a legacy config first
	lcfg, err := NewLegacyCollectorConfig(data)
	// If it's a legacy config, convert to standard config
	if err == nil {
		cfg, err := lcfg.ToDefaultCollectorConfig(*dstPort)
		// If that fails, bubble up
		if err != nil {
			return err
		}
		// Save it and we're done
		c.cfg = cfg
		return nil
	}
	// If it's not a legacy config, handle it like a standard one
	cfg, err := NewCollectorConfig(data)
	// If that fails, bubble up
	if err != nil {
		return err
	}
	// Save it and we're done
	c.cfg = cfg
	return nil
}

// SetupAPI creates and performs initial setup of the API based on the config.
func (c *Collector) SetupAPI() {
	LogInfo("Setting up API")
	// If we don't have a Summarizer, create one
	if c.s == nil {
		c.SetupSummarizer()
	}
	c.api = NewAPI(c.s, c.ts, c.cfg.API.Bind)
}

// SetupTagSet loads the tags for targets, based on the config, that will be
// applied to summarized results.
func (c *Collector) SetupTagSet() {
	LogInfo("Setting up tag set")
	c.ts = c.cfg.Targets.TagSet()
}

// SetupTestRunner takes parameters from the loaded config, and creates the
// specified TestConfig.
func (c *Collector) SetupTestRunner(test TestConfig) {
	rl := c.createRateLimiter(test.RateLimit)
	runner := NewTestRunner(c.cbc, rl)
	// TODO(nwinemiller): This could hit a runtime error if the TargetSet name
	// doesn't exist. So might want to break this into two parts.
	targets, err := c.cfg.Targets[test.Targets].ListResolvedTargets()
	if err != nil {
		HandleFatalErrorMsg(err, "failed to resolve targets")
	}
	runner.Set(targets)
	c.createPortGroupOnRunner(runner, test.PortGroup)
	c.runners = append(c.runners, runner)
}

// SetupTestRunners creates all the `tests` that are defined in the config.
func (c *Collector) SetupTestRunners() {
	LogInfo("Setting up test runners")
	// Don't recreate the channel on reload, only create once
	if c.cbc == nil {
		c.cbc = make(chan *Probe, DEFAULT_CHANNEL_SIZE)
	}
	// If there are already test runners, they should be removed
	if len(c.runners) > 0 {
		LogInfo("Found old test runners. Stopping and purging.")
		for _, runner := range c.runners {
			runner.Stop()
		}
		// Clear out the slice
		c.runners = nil
	}
	for _, test := range c.cfg.Tests {
		c.SetupTestRunner(test)
	}
}

// createRateLimiter creates a TestRunner compliant RateLimter based on the
// config for the named rate limiter.
func (c *Collector) createRateLimiter(name string) *rate.Limiter {
	rlConfig := c.cfg.RateLimits[name]
	rl := rate.NewLimiter(rate.Limit(rlConfig.CPS), int(rlConfig.CPS))
	return rl
}

// createPortOnRunner creates a port on the provided TestRunner based on the
// provided PortConfig.
func (c *Collector) createPortOnRunner(runner *TestRunner, p PortConfig) {
	timeout := time.Duration(p.Timeout) * time.Millisecond
	runner.AddNewPort(
		fmt.Sprintf("%v:%v", p.IP, p.Port),
		byte(p.Tos),
		timeout,
		timeout,
		timeout,
	)
}

// createPortGroupOnRunner creates the named port group from the config on the
// provided TestRunner instance.
func (c *Collector) createPortGroupOnRunner(runner *TestRunner, name string) {
	pg := c.cfg.PortGroups[name]
	for _, pgc := range pg {
		for i := int64(0); i < pgc.Count; i++ {
			c.createPortOnRunner(runner, c.cfg.Ports[pgc.Port])
		}
	}
}

// SetupSummarizer creates the Summarizer and ResultHandlers that will
// summarize and save the test results, based on the config.
func (c *Collector) SetupSummarizer() {
	LogInfo("Setting up summarizer")
	// Setup the summarizer and result handlers
	resultChan := make(chan *Result, DEFAULT_CHANNEL_SIZE)
	c.s = NewSummarizer(
		resultChan,
		time.Duration(c.cfg.Summarization.Interval)*time.Second,
	)
	c.setupResultHandlers(resultChan)
}

// setupResultHandlers creates number of ResultHandlers defined by the config.
func (c *Collector) setupResultHandlers(resultChan chan *Result) {
	LogInfo(fmt.Sprintf("Setting up %d result handlers", c.cfg.Summarization.Handlers))
	for i := int64(0); i < c.cfg.Summarization.Handlers; i++ {
		rh := NewResultHandler(c.cbc, resultChan)
		c.rh = append(c.rh, rh)
	}
}

// Setup is a generally wrapper around all of the other Setup* functions.
func (c *Collector) Setup() {
	// Ordering is important here, as some of these depend on elements
	// setup earlier in the process.
	LogInfo("Setting up collector")
	c.LoadConfig()
	c.SetupTagSet()
	c.SetupTestRunners()
	c.SetupSummarizer()
	c.SetupAPI()
	LogInfo("Collector setup complete")
}

// Reload causes the config to be reread, and test runners recreated
func (c *Collector) Reload() {
	LogInfo("Reloading collector")
	// This should be an atomic operation, so no prep needed
	c.LoadConfig()
	// Same here
	c.SetupTagSet()
	// This will purge existing test runners and rebuild
	c.SetupTestRunners()
	// The summarizer and API should be untouched though
	// We just need to start all the new test runners
	// TODO(nwinemiller): This is redundant with part of Run() and
	//             could be reorganized.
	LogInfo("Starting new test runners")
	for _, runner := range c.runners {
		runner.Run()
	}
	// Update the TagSet on the API to reflect the new config
	// TODO(nwinemiller): This merges the new TagSet with the existing one to address the case
	//   where outstanding test results are for a host that is no longer in the config.
	//   So if we get rid of the existing tag info, when that one gets to the API,
	//   it'll have no tags. So this continues updating the existing tagset with
	//   the latest information each time, but keeping old data around.
	//   This definitely isn't ideal, but sorting out what to keep or not is
	//   non-trivial. So keep this as an improvement for the refactor.
	LogInfo("Updating TagSet on API")
	c.api.MergeUpdateTagSet(c.ts)
	LogInfo("Collector reload complete")
}

// Run starts all of the components of the collector and begins testing.
func (c *Collector) Run() {
	LogInfo("Starting Collector")
	// Start the API
	c.api.Run()
	// Start the Summarizer
	c.s.Run()
	// Start the ResultHandlers
	for _, rh := range c.rh {
		rh.Run()
	}
	// Start the TestRunners
	for _, runner := range c.runners {
		runner.Run()
	}
	LogInfo("All Collector components running")
}

// Stop will signal all collector components to stop.
func (c *Collector) Stop() {
	LogInfo("Stopping Collector")
	// Stop the TestRunners
	for _, runner := range c.runners {
		runner.Stop()
	}
	// Stop the ResultHandlers
	for _, rh := range c.rh {
		rh.Stop()
	}
	// Stop the Summarizer
	c.s.Stop()
	// Stop the API
	c.api.Stop()
	LogInfo("All Collector components signaled to stop")
}
