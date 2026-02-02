> **Note:** This is a fork of the original [Dropbox LLAMA](https://github.com/dropbox/llama) project.
> It has been modified to export Prometheus metrics instead of InfluxDB and includes
> Docker deployment support.

# LLAMA

LLAMA (Loss and LAtency MAtrix) is a library for testing and measuring network loss and latency between distributed endpoints.

It does this by sending UDP datagrams/probes from **collectors** to **reflectors** and measuring how long it takes for them to return, if they return at all. UDP is used to provide ECMP hashing over multiple paths (a win over ICMP) without the need for setup/teardown and per-packet granularity (a win over TCP).

## Why Is This Useful

[Black box testing](https://en.wikipedia.org/wiki/Black-box_testing) is critical to the successful monitoring and operation of a network. While collection of metrics from network devices can provide greater detail regarding known issues, they don't always provide a complete picture and can provide an overwhelming number of metrics. Black box testing with LLAMA doesn't care how the network is structured, only if it's working. This data can be used for building KPIs, observing big-picture issues, and guiding investigations into issues with unknown causes by quantifying which flows are/aren't working.

Network operators have found this useful on multiple occasions for gauging the impact of network issues on internal traffic, identifying the scope of impact, and locating issues for which they had no other metrics (internal hardware failures, circuit degradations, etc).

**Even if you operate entirely in the cloud** LLAMA can help identify reachability and network health issues between and within regions/zones.

## Prometheus Metrics

The collector and reflector both expose Prometheus metrics on their `/metrics` endpoints.

### Collector Metrics

The collector exposes the following metrics on port 5200:

- **`llama_packet_loss_percentage`** (Gauge) - Packet loss percentage for a given measurement period.
- **`llama_packets_sent`** (Gauge) - Number of packets sent for a given measurement period.
- **`llama_packets_lost`** (Gauge) - Number of packets lost for a given measurement period.
- **`llama_rtt`** (Gauge) - Average round-trip time (RTT) for packets sent during a given measurement period.

### Reflector Metrics

The reflector exposes the following metrics on port 8200:

| Metric | Type | Description |
|--------|------|-------------|
| `llama_reflector_packets_received_total` | Counter | Total UDP packets received by the reflector |
| `llama_reflector_packets_reflected_total` | Counter | Packets successfully reflected back to sender |
| `llama_reflector_packets_bad_data_total` | Counter | Malformed/unparseable packets received |
| `llama_reflector_packets_throttled_total` | Counter | Packets dropped due to rate limiting |
| `llama_reflector_tos_changes_total` | Counter | ToS bit changes on the socket |
| `llama_reflector_up` | Gauge | Health status: 1 if running, 0 if stopped |

### Labels

Collector metrics include the following labels:

- **`src_ip`** - Source IP address of the collector.
- **`dst_ip`** - Destination IP address of the reflector.
- **`src_hostname`** - Source hostname (from config tags).
- **`dst_hostname`** - Destination hostname (from config tags).

### Example Query

Query average packet loss across all destinations:
```promql
avg by (dst_hostname, dst_ip) (llama_packet_loss_percentage)
```

Query average RTT for a specific destination:
```promql
llama_rtt{dst_hostname="server-1"}
```

Alert on high packet loss:
```promql
avg by (dst_hostname) (llama_packet_loss_percentage) > 5
```

## Architecture

- **Reflector** - Lightweight daemon for receiving probes and sending them back to their source.
- **Collector** - Sends probes to reflectors on potentially multiple ports, records results, and exposes Prometheus metrics via `/metrics` endpoint.
- **Prometheus** - External Prometheus server scrapes metrics from the collector's `/metrics` endpoint for monitoring and alerting.

## Quick Start

If you're looking to get started quickly with a basic setup that doesn't involve special integrations or customization, this should get you going. This assumes you have Prometheus configured to scrape from the collector.

### Local Development

In your Go development environment, in separate windows:

- `go run github.com/nsw3550/llama/cmd/reflector`
- `go run github.com/nsw3550/llama/cmd/collector -llama.config configs/simple_example.yaml`

The collector will expose metrics on `http://localhost:5200/metrics` for Prometheus to scrape.

### Prometheus Configuration

Add a scrape configuration to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'llama'
    static_configs:
      - targets: ['localhost:5200']
    scrape_interval: 30s  # Align with collector summarization interval
```

### Docker Deployment

Pre-built Docker images are available on Docker Hub:
- `tenkenx/llama-collector`
- `tenkenx/llama-reflector`

#### Run Pre-built Images

**Run Reflector:**
```bash
docker run -d \
  --name llama-reflector \
  -p 8100:8100 \
  -p 8200:8200 \
  tenkenx/llama-reflector
```

**Run Collector:**
```bash
docker run -d \
  --name llama-collector \
  -v /path/to/config.yaml:/etc/llama/config.yaml \
  -p 5200:5200 \
  tenkenx/llama-collector
```

#### Build from Source (Single Architecture)

**Build Reflector:**
```bash
docker build -t llama-reflector -f cmd/reflector/Dockerfile .
```

**Build Collector:**
```bash
docker build -t llama-collector -f cmd/collector/Dockerfile .
```

#### Build for Multiple Architectures (buildx)

To build images that support both `amd64` and `arm64` architectures, use `docker buildx`:

**Prerequisites:**
```bash
# Create a new builder instance
docker buildx create --name multiarch-builder --use

# Inspect builder and bootstrap
docker buildx inspect --bootstrap
```

**Build Reflector for amd64 and arm64:**
```bash
# Build and load for local platform (faster for testing)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file cmd/reflector/Dockerfile \
  --tag llama-reflector:latest \
  --load \
  .

# Or build and push directly to registry
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file cmd/reflector/Dockerfile \
  --tag your-registry/llama-reflector:latest \
  --push \
  .
```

**Build Collector for amd64 and arm64:**
```bash
# Build and load for local platform
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file cmd/collector/Dockerfile \
  --tag llama-collector:latest \
  --load \
  .

# Or build and push directly to registry
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file cmd/collector/Dockerfile \
  --tag your-registry/llama-collector:latest \
  --push \
  .
```

**Notes:**
- Use `--load` flag to build for your current platform only (required if running locally)
- Use `--push` flag to build all platforms and push to a Docker registry
- You can also specify a single platform if needed: `--platform linux/arm64`
- The images use multi-arch base images, so they will run on both architectures

### Production Deployment

For production deployment on separate machines/instances:

- Reflector: `reflector -port <port>` to start the reflector listening on a non-default port.
- Collector: `collector -llama.config <config>` where the config is a YAML configuration based on one of the examples under `configs/`.

Configure Prometheus to scrape the collector's `/metrics` endpoint from the API port (default: 5200).

## Ongoing Development

This is a fork of the original Dropbox LLAMA project. The original was built during a [Dropbox Hack Week](https://www.theverge.com/2014/7/24/5930927/why-dropbox-gives-its-employees-a-week-to-do-whatever-they-want). This fork is currently in early development with significant changes including migration to Prometheus metrics and modernized dependencies. The API and config format may continue to evolve.

## Contributing

This is a very early stage project. Contributions are welcome, but please check with the maintainer first before submitting pull requests. We appreciate your interest in improving LLAMA!

## Acknowledgements/References

* Inspired by: <https://www.youtube.com/watch?v=N0lZrJVdI9A>
    * With slides: <https://www.nanog.org/sites/default/files/Lapukhov_Move_Fast_Unbreak.pdf>
* Concepts borrowed from: <https://github.com/facebook/UdpPinger/>
* Looking for the legacy Python version?: https://github.com/dropbox/llama-archive
