# oci-prometheus-sd-proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/amaanx86/oci-prometheus-sd-proxy)](https://goreportcard.com/report/github.com/amaanx86/oci-prometheus-sd-proxy)
[![GitHub Release](https://img.shields.io/github/v/release/amaanx86/oci-prometheus-sd-proxy)](https://github.com/amaanx86/oci-prometheus-sd-proxy/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Image](https://img.shields.io/badge/Docker-ghcr.io-blue?logo=docker)](https://github.com/amaanx86/oci-prometheus-sd-proxy/pkgs/container/oci-prometheus-sd-proxy)

<img width="469" height="277" alt="OCI Prometheus SD Proxy" src="https://github.com/user-attachments/assets/333a7c32-93bd-4ad9-aea3-aea2d6a66a65" />

A lightweight Go service that implements the [Prometheus HTTP Service Discovery](https://prometheus.io/docs/prometheus/latest/http_sd/) API for [Oracle Cloud Infrastructure](https://www.oracle.com/cloud/). It dynamically discovers compute instances across multiple OCI tenancies and compartments, filters them by tag, and returns their metadata in Prometheus-compatible HTTP SD format.

## Architecture

![oci-sd-proxy-arch](https://github.com/user-attachments/assets/a7d87901-1e67-4016-92b6-df66f5603b28)

Multiple Prometheus replicas query the service discovery endpoint, which fetches instance data from multiple OCI tenancies in parallel and returns rich metadata for relabeling.

## Quick Start

### Docker

```bash
docker run -d \
  -e SERVER_TOKEN="$(openssl rand -hex 32)" \
  -v /path/to/config.yaml:/etc/oci-sd/config.yaml:ro \
  -v /path/to/oci/keys:/etc/oci-sd/keys:ro \
  -p 8080:8080 \
  ghcr.io/amaanx86/oci-prometheus-sd-proxy:latest
```

### Docker Compose

```bash
cd deploy/docker
cp .env.example .env
cp config.yaml.example config.yaml
cp ~/.oci/api_key.pem oci-keys/
docker-compose -f docker-compose-production.yml up -d
```

### Prometheus Config

```yaml
scrape_configs:
  - job_name: oci_instances
    http_sd_configs:
      - url: 'http://oci-sd-proxy:8080/v1/targets'
        authorization:
          type: Bearer
          credentials: 'YOUR_TOKEN'
        refresh_interval: 60s
    scrape_interval: 30s
    scrape_timeout: 10s
    metrics_path: /metrics
    relabel_configs:
      - source_labels: [__meta_oci_instance_name]
        target_label: instance
      - source_labels: [__meta_oci_tenancy_name]
        target_label: tenancy
      - source_labels: [__meta_oci_compartment_name]
        target_label: compartment
      - source_labels: [__meta_oci_region]
        target_label: region
      - source_labels: [__meta_oci_availability_domain]
        target_label: availability_domain
      - source_labels: [__meta_oci_instance_shape]
        target_label: shape
```

## Full Documentation

Complete documentation available at: **https://oci-prometheus-sd-proxy.readthedocs.io/**

- Installation & setup
- Configuration reference
- OCI API permissions
- Prometheus integration
- Security best practices
- API reference

## Windows Instances - Port Selection

The proxy needs to know whether an instance is Linux or Windows to pick the right exporter port:

| OS | Default Port | Exporter |
|----|-------------|----------|
| Linux | `9100` | [node_exporter](https://github.com/prometheus/node_exporter) |
| Windows | `9182` | [windows_exporter](https://github.com/prometheus-community/windows_exporter) |

**Detection order** (first match wins):

1. Freeform tag `os` = `windows` on the OCI instance
2. VM display name contains `win` (e.g. `win-server-01`, `windows-web`)
3. Everything else defaults to port `9100`

**Recommended approach for Windows instances:**

Set the freeform tag `os = windows` on the OCI instance, or make sure `win` appears in the VM display name.

When installing `windows_exporter` via the MSI installer, configure it to listen on port `9182` (the default). If you prefer port `9100`, set that in the MSI installer and update `windows_port` in the proxy config to match - or simply leave both at their defaults and rely on the tag/name detection above.

> **Note:** If a Windows VM has no `os` tag and no `win` in its name, the proxy will target it on port `9100`. In that case, either set the tag, rename the VM, or configure `windows_exporter` to listen on port `9100` during MSI installation.

## Features

- **Multi-tenancy**: Discover instances across any number of OCI tenancies
- **Tag-based filtering**: Only scrape instances with configured tags
- **Rich labels**: Tenancy, compartment, shape, region, and all custom tags
- **Fast discovery**: Parallel compartment scanning with caching
- **Rate limiting**: Proactive token bucket + reactive retry policy prevent 429 errors
- **Secure**: Bearer token auth, distroless image, read-only config mounts
- **Production-ready**: JSON logging, health probes, configurable refresh

## API Endpoints

- **GET `/v1/targets`** - Prometheus HTTP SD endpoint (requires Bearer token)
- **GET `/healthz`** - Liveness probe
- **GET `/readyz`** - Readiness probe

## Maintainer

**[Amaan Ul Haq Siddiqui](https://github.com/amaanx86)**
- Email: amaanulhaq.s@outlook.com
- LinkedIn: [amaanulhaqsiddiqui](https://www.linkedin.com/in/amaanulhaqsiddiqui/)

## License

MIT - See [LICENSE](LICENSE) for details.
