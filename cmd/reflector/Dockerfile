FROM golang:1.24 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags=netgo -o /app/llama_reflector ./cmd/reflector

FROM gcr.io/distroless/base-debian11 AS release-stage
WORKDIR /
COPY --from=build-stage /app/llama_reflector /llama_reflector
USER nonroot:nonroot
EXPOSE 8100 8200
ENTRYPOINT ["/llama_reflector"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget --no-verbose --tries=1 --spider http://localhost:8200/status || exit 1