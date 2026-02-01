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
EXPOSE 8100
ENTRYPOINT ["/llama_reflector"]