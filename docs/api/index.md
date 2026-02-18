# API Reference

This section documents the Go API for UDProbe. The library provides packages for building custom collectors and reflectors.

## Package: udprobe

### Core Types

#### CollectorConfig

```go
type CollectorConfig struct {
    Summarization SummarizationConfig `yaml:"summarization"`
    API           APIConfig           `yaml:"api"`
    Ports         PortsConfig         `yaml:"ports"`
    PortGroups    PortGroupsConfig    `yaml:"port_groups"`
    RateLimits    RateLimitsConfig    `yaml:"rate_limits"`
    Tests         TestsConfig         `yaml:"tests"`
    Targets       TargetsConfig       `yaml:"targets"`
}
```

The main configuration structure for the collector. Contains all sub-configurations for ports, targets, rate limits, and summarization.

#### TargetConfig

```go
type TargetConfig struct {
    IP   string `yaml:"ip"`
    Port int64  `yaml:"port"`
    Tags Tags   `yaml:"tags"`
}
```

Defines a single target reflector endpoint with IP, port, and associated tags for metrics labeling.

#### PortConfig

```go
type PortConfig struct {
    IP      string `yaml:"ip"`
    Port    int64  `yaml:"port"`
    Tos     int64  `yaml:"tos"`
    Timeout int64  `yaml:"timeout"`
}
```

Configuration for a UDP port used for sending probes.

#### PortGroupConfig

```go
type PortGroupConfig struct {
    Port  string `yaml:"port"`
    Count int64  `yaml:"count"`
}
```

Defines a group of identical ports for parallel probe sending.

#### RateLimitConfig

```go
type RateLimitConfig struct {
    CPS float64 `yaml:"cps"`
}
```

Rate limiting configuration (cycles per second).

#### TestConfig

```go
type TestConfig struct {
    Targets   string `yaml:"targets"`
    PortGroup string `yaml:"port_group"`
    RateLimit string `yaml:"rate_limit"`
}
```

Combines targets, port groups, and rate limits into a test definition.

### Key Functions

#### NewCollectorConfig

```go
func NewCollectorConfig(data []byte) (*CollectorConfig, error)
```

Creates a CollectorConfig from YAML byte data.

#### NewDefaultCollectorConfig

```go
func NewDefaultCollectorConfig() (*CollectorConfig, error)
```

Returns a sensible default configuration.

### Collector

```go
type Collector struct {
    cfg *CollectorConfig
    ts  TagSet
    api *API
    runners []*TestRunner
    cbc chan *InFlightProbe
    s   *Summarizer
    rh  []*ResultHandler
}
```

The main collector structure that sends probes and tracks results.

#### Collector Methods

- `LoadConfig()` - Loads configuration from file or defaults
- `Run()` - Starts the collector (non-blocking)
- `RunForever()` - Starts the collector (blocking)

### API

```go
type API struct {
    summarizer *Summarizer
    server     *http.Server
    ts         TagSet
    handler    *http.ServeMux
    mutex      sync.RWMutex
}
```

HTTP API server for exposing metrics.

#### API Methods

- `PromHandler()` - Returns handler for Prometheus metrics endpoint
- `StatusHandler()` - Health check handler (returns 200 OK)
- `Stop()` - Shuts down the API server
- `Run()` - Starts API server (non-blocking)
- `RunForever()` - Starts API server (blocking)

### Reflector Functions

#### Reflect

```go
func Reflect(conn *net.UDPConn, rl *rate.Limiter)
```

Main reflector loop. Listens on the provided UDP connection and reflects probes back to senders.

#### Receive

```go
func Receive(data []byte, oob []byte, conn *net.UDPConn) ([]byte, []byte, *net.UDPAddr)
```

Receives a UDP packet and returns the data, control message, and source address.

#### Send

```go
func Send(data []byte, tos byte, conn *net.UDPConn, addr *net.UDPAddr)
```

Sends a UDP packet to the specified address with the given Type of Service byte.

### HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `/status` | Health check - returns "ok" |
| `/metrics` | Prometheus metrics endpoint |

## Related Documentation

- [Configuration](../configuration.md) - Configuration file format
- [Architecture](../architecture.md) - System architecture and design
