# Architecture

UDProbe consists of two main components: the **Reflector** and the **Collector**. These work together to measure network loss and latency between distributed endpoints.

## Overview

```
+--------+                      +-----------+
|        |   UDP Probes         |           |
|        |  =============>       |           |
|        |                      |           |
|  COL   |   Reflected Probes   |  REFLECTOR|
| LECTOR |  <=============      |           |
|        |                      |           |
|        |   Prometheus Metrics |           |
+--------+---------------------> |           |
        :5200/metrics           |           |
                                +-----------+
                                      :8100
```

## Components

### Reflector

The reflector is a lightweight daemon that receives UDP probes and immediately sends them back to their source. It's designed to have minimal overhead and fast response times.

**Responsibilities:**
- Listen for incoming UDP probes on a configurable port (default 8100)
- Unmarshal the probe to validate it's a valid UDProbe packet
- Add a receive timestamp to the probe
- Re-marshal and reflect the probe back to the sender
- Expose Prometheus metrics on port 8200

**Prometheus Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `udprobe_reflector_packets_received_total` | Counter | Total UDP packets received |
| `udprobe_reflector_packets_reflected_total` | Counter | Packets successfully reflected |
| `udprobe_reflector_packets_bad_data_total` | Counter | Malformed packets received |
| `udprobe_reflector_packets_throttled_total` | Counter | Packets dropped due to rate limiting |
| `udprobe_reflector_tos_changes_total` | Counter | ToS bit changes on the socket |
| `udprobe_reflector_up` | Gauge | Health status |

### Collector

The collector is responsible for sending probes, tracking their responses, and exposing metrics for Prometheus to scrape.

**Responsibilities:**
- Load and parse configuration (targets, ports, rate limits)
- Send UDP probes to configured reflectors at defined rates
- Track in-flight probes and their send times
- Receive reflected probes and calculate round-trip times
- Summarize results at configurable intervals
- Expose Prometheus metrics on port 5200

**Data Flow:**

1. **Configuration Loading** - Collector reads YAML config defining targets, ports, and rate limits
2. **TestRunner Creation** - Creates TestRunner instances for each configured test
3. **Probe Sending** - TestRunner sends probes at configured rate using multiple ports
4. **Probe Tracking** - Sent probes are tracked with timestamps in an in-flight map
5. **Reflection Receipt** - Returned probes are matched with sent probes
6. **Summarization** - Results are aggregated and metrics updated at configured intervals

**Prometheus Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `udprobe_packet_loss_percentage` | Gauge | Packet loss percentage |
| `udprobe_packets_sent` | Gauge | Packets sent in period |
| `udprobe_packets_lost` | Gauge | Packets lost in period |
| `udprobe_rtt` | Gauge | Average RTT in milliseconds |

**Metric Labels:**

- `src_ip` - Source IP address of the collector
- `dst_ip` - Destination IP address of the reflector
- `src_hostname` - Source hostname (from config tags)
- `dst_hostname` - Destination hostname (from config tags)

## Protocol

UDProbe uses Protocol Buffers for packet serialization. The probe message contains:

- **UUID** - Unique identifier for tracking
- **Sent** - Timestamp when probe was sent (Unix nanoseconds)
- **Rcvd** - Timestamp when probe was received by reflector (Unix nanoseconds)
- **Tos** - Type of Service byte

## Design Decisions

### Why UDP?

- **ECMP Hashing** - UDP allows for ECMP path hashing, useful for testing multiple network paths
- **Per-packet Granularity** - Unlike TCP, each UDP packet is independent
- **No Setup/Teardown** - No handshake overhead, faster probing
- **ICMP Limitations** - ICMP can be rate-limited or routed differently than data traffic

### Why Black Box Testing?

Black box network testing doesn't require knowledge of the network topology. It simply tests whether traffic can get from point A to point B and back. This is useful for:

- Identifying network-wide issues
- Measuring impact of failures
- Validating network changes
- Providing KPIs for network health
