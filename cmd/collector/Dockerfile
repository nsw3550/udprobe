FROM golang:1.24 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/udprobe_collector ./cmd/collector

FROM gcr.io/distroless/base-debian11 AS release-stage
WORKDIR /
COPY --from=build-stage /app/udprobe_collector /udprobe_collector
USER nonroot:nonroot
EXPOSE 5200
ENTRYPOINT ["/udprobe_collector", "-udprobe.config=/etc/udprobe/config.yaml"]