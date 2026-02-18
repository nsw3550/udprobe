# Configuration

UDProbe uses YAML configuration files to define targets, port groups, rate limits, and test parameters.

## Configuration Format

There are two configuration formats supported:

1. **Simple format** - Legacy format with basic IP to tags mapping
2. **Full format** - Complete configuration with all options

## Simple Format

The simple format maps IP addresses to tags:

```yaml
<ip-address>:
    <tag>: <value>
```

Example:

```yaml
127.0.0.1:
    dst_hostname: localhost
    dst_region: west
127.0.0.2:
    dst_hostname: server-2
    dst_region: east
```

## Full Format

The full configuration format provides fine-grained control over the collector's operation.

### Top-Level Keys

| Key | Type | Description |
|-----|------|-------------|
| `summarization` | object | Controls how test results are aggregated |
| `api` | object | Controls the REST API server |
| `ports` | object | Port configuration definitions |
| `port_groups` | object | Groupings of ports for parallel testing |
| `rate_limits` | object | Rate limiting configuration |
| `tests` | array | Test definitions combining other config |
| `targets` | object | Target reflector endpoints |

### Summarization

Controls how often test results are aggregated:

```yaml
summarization:
    interval:   30    # Summary interval in seconds
    handlers:   2     # Number of result handlers
```

| Field | Type | Description |
|-------|------|-------------|
| `interval` | int | How often to summarize results (seconds) |
| `handlers` | int | Number of result handler goroutines |

### API

Controls the HTTP API server:

```yaml
api:
    bind:   0.0.0.0:5200    # Bind address and port
```

| Field | Type | Description |
|-----|------|-------------|
| `bind` | string | Address and port to listen on |

### Ports

Defines UDP port configurations for sending probes:

```yaml
ports:
    default:
        ip:         0.0.0.0       # Source IP (0.0.0.0 = any)
        port:       0              # Port (0 = auto-select)
        tos:        0              # Type of Service byte
        timeout:    1000           # Timeout in milliseconds
```

| Field | Type | Description |
|-----|------|-------------|
| `ip` | string | Source IP address |
| `port` | int | Source port (0 for OS-assigned) |
| `tos` | int | Type of Service byte value |
| `timeout` | int | Socket timeout in milliseconds |

### Port Groups

Groups ports together for parallel testing:

```yaml
port_groups:
    default:
        - port:     default
          count:    4
```

| Field | Type | Description |
|-----|------|-------------|
| `port` | string | Reference to a port config |
| `count` | int | Number of parallel ports |

### Rate Limits

Defines rate limiting for probes:

```yaml
rate_limits:
    default:
        cps:    4.0    # Cycles per second
```

| Field | Type | Description |
|-----|------|-------------|
| `cps` | float | Probes per second (per port in group) |

### Tests

Defines test configurations:

```yaml
tests:
    - targets:      default
      port_group:   default
      rate_limit:   default
```

| Field | Type | Description |
|-----|------|-------------|
| `targets` | string | Reference to targets config |
| `port_group` | string | Reference to port group |
| `rate_limit` | string | Reference to rate limit |

### Targets

Defines reflector endpoints to test:

```yaml
targets:
    default:
        - ip:   127.0.0.1
          port: 8100
          tags:
            dst_hostname: localhost
            dst_region: west
```

| Field | Type | Description |
|-----|------|-------------|
| `ip` | string | Target IP address |
| `port` | int | Target port |
| `tags` | object | Key-value pairs for metrics labeling |

## Complete Example

```yaml
summarization:
    interval:   30
    handlers:   2

api:
    bind:   0.0.0.0:5200

ports:
    default:
        ip:         0.0.0.0
        port:       0
        tos:        0
        timeout:    1000

port_groups:
    default:
        - port:     default
          count:    4

rate_limits:
    default:
        cps:    4.0

tests:
    - targets:      default
      port_group:   default
      rate_limit:   default

targets:
    default:
        - ip:   192.168.1.1
          port: 8100
          tags:
            dst_hostname: reflector-1
            dst_region: west
        - ip:   192.168.1.2
          port: 8100
          tags:
            dst_hostname: reflector-2
            dst_region: east
```

## Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `udprobe.config` | Path to configuration file | (none) |
| `udprobe.dst-port` | Target port (legacy config only) | 8100 |
