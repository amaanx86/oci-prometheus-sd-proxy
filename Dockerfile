FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache dependency downloads separately from source build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /bin/oci-sd-proxy \
    ./cmd/server

#FROM gcr.io/distroless/static-debian12:nonroot

# Metadata labels
LABEL org.opencontainers.image.title="oci-prometheus-sd-proxy" \
      org.opencontainers.image.description="Prometheus HTTP Service Discovery proxy for Oracle Cloud Infrastructure" \
      org.opencontainers.image.url="https://github.com/amaanx86/oci-prometheus-sd-proxy" \
      org.opencontainers.image.source="https://github.com/amaanx86/oci-prometheus-sd-proxy" \
      org.opencontainers.image.documentation="https://github.com/amaanx86/oci-prometheus-sd-proxy/tree/main/docs" \
      org.opencontainers.image.authors="Amaan Ul Haq Siddiqui <amaanulhaq.s@outlook.com>" \
      org.opencontainers.image.vendor="amaanx86" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.base.name="gcr.io/distroless/static-debian12:nonroot"

COPY --from=builder /bin/oci-sd-proxy /oci-sd-proxy

# OCI PEM keys and config.yaml are expected to be mounted at runtime
VOLUME ["/etc/oci-sd"]

EXPOSE 8080

ENTRYPOINT ["/oci-sd-proxy"]
