# UDProbe

UDProbe (mix of UDP and Probe) is a library for testing and measuring network loss and latency between distributed endpoints.

It does this by sending UDP datagrams/probes from **collectors** to **reflectors** and measuring how long it takes for them to return, if they return at all. UDP is used to provide ECMP hashing over multiple paths (a win over ICMP) without the need for setup/teardown and per-packet granularity (a win over TCP).

## Why Is This Useful

[Black box testing](https://en.wikipedia.org/wiki/Black-box_testing) is critical to the successful monitoring and operation of a network. While collection of metrics from network devices can provide greater detail regarding known issues, they don't always provide a complete picture and can provide an overwhelming number of metrics. Black box testing with UDProbe doesn't care how the network is structured, only if it's working. This data can be used for building KPIs, observing big-picture issues, and guiding investigations into issues with unknown causes by quantifying which flows are/aren't working.

Network operators often find this useful for gauging the impact of network issues on internal traffic, identifying the scope of impact, and locating issues for which they had no other metrics (internal hardware failures, circuit degradations, etc).

**Even if you operate entirely in the cloud** UDProbe can help identify reachability and network health issues between and within regions/zones.

## Quick Start

### Local Development

```bash
# Run the reflector
go run github.com/nsw3550/udprobe/cmd/reflector

# Run the collector (in a separate terminal)
go run github.com/nsw3550/udprobe/cmd/collector -udprobe.config configs/simple_example.yaml
```

The collector will expose metrics on `http://localhost:5200/metrics` for Prometheus to scrape.

### Docker Deployment

```bash
# Run Reflector
docker run -d \
  --name udprobe-reflector \
  -p 8100:8100 \
  -p 8200:8200 \
  tenkenx/udprobe-reflector

# Run Collector
docker run -d \
  --name udprobe-collector \
  -v /path/to/config.yaml:/etc/udprobe/config.yaml \
  -p 5200:5200 \
  tenkenx/udprobe-collector
```

## Prometheus Metrics

### Collector Metrics

The collector exposes the following metrics on port 5200:

| Metric | Type | Description |
|--------|------|-------------|
| `udprobe_packet_loss_percentage` | Gauge | Packet loss percentage for a given measurement period |
| `udprobe_packets_sent` | Gauge | Number of packets sent for a given measurement period |
| `udprobe_packets_lost` | Gauge | Number of packets lost for a given measurement period |
| `udprobe_rtt` | Gauge | Average round-trip time (RTT) for packets sent during a given measurement period |

### Reflector Metrics

The reflector exposes the following metrics on port 8200:

| Metric | Type | Description |
|--------|------|-------------|
| `udprobe_reflector_packets_received_total` | Counter | Total UDP packets received by the reflector |
| `udprobe_reflector_packets_reflected_total` | Counter | Packets successfully reflected back to sender |
| `udprobe_reflector_packets_bad_data_total` | Counter | Malformed/unparseable packets received |
| `udprobe_reflector_packets_throttled_total` | Counter | Packets dropped due to rate limiting |
| `udprobe_reflector_tos_changes_total` | Counter | ToS bit changes on the socket |
| `udprobe_reflector_up` | Gauge | Health status: 1 if running, 0 if stopped |

## Links

- [GitHub Repository](https://github.com/nsw3550/udprobe)
- [Installation Guide](installation.md)
- [Configuration Reference](configuration.md)
- [Architecture Overview](architecture.md)
- [API Reference](api/index.md)
