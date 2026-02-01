FROM golang:1.24 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/llama_collector ./cmd/collector

FROM gcr.io/distroless/base-debian11 AS release-stage
WORKDIR /
COPY --from=build-stage /app/llama_collector /llama_collector
USER nonroot:nonroot
EXPOSE 5200
ENTRYPOINT ["/llama_collector", "-llama.config=/etc/llama/config.yaml"]