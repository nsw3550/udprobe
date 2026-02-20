# Installation

## Prerequisites

- Go 1.24 or later (if building from source)
- Docker (for containerized deployment)
- Prometheus (for metrics collection)

## Docker Deployment (Easiest)

### Pre-built Images

Pre-built Docker images are available on Docker Hub:

- `tenkenx/udprobe-collector`
- `tenkenx/udprobe-reflector`

### Run Pre-built Images

**Run Reflector:**

```bash
docker run -d \
  --name udprobe-reflector \
  -p 8100:8100 \
  -p 8200:8200 \
  tenkenx/udprobe-reflector
```

**Run Collector:**

```bash
docker run -d \
  --name udprobe-collector \
  -v /path/to/config.yaml:/etc/udprobe/config.yaml \
  -p 5200:5200 \
  tenkenx/udprobe-collector
```

## From Source

### Clone the Repository

```bash
git clone https://github.com/nsw3550/udprobe.git
cd udprobe
```

### Build the Binaries

```bash
# Build collector
go build -o udprobe-collector ./cmd/collector

# Build reflector
go build -o udprobe-reflector ./cmd/reflector
```

### Run the Binaries

```bash
# Run reflector
./udprobe-reflector

# Run collector (requires configuration)
./udprobe-collector -udprobe.config /path/to/config.yaml
```


## Building Docker Images from Source

**Build Reflector:**

```bash
docker build -t udprobe-reflector -f cmd/reflector/Dockerfile .
```

**Build Collector:**

```bash
docker build -t udprobe-collector -f cmd/collector/Dockerfile .
```

### Multi-Architecture Docker Builds

To build images for both `amd64` and `arm64` architectures:

```bash
# Create builder
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

# Build and load locally
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file cmd/reflector/Dockerfile \
  --tag udprobe-reflector:latest \
  --load \
  .

# Or build and push to registry
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file cmd/reflector/Dockerfile \
  --tag your-registry/udprobe-reflector:latest \
  --push \
  .
```

