# OCI SD Proxy - Docker Compose for Docker Deployment

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/oci-prometheus-sd-proxy)](https://artifacthub.io/packages/search?repo=oci-prometheus-sd-proxy)
[![GitHub Release](https://img.shields.io/github/v/release/amaanx86/oci-prometheus-sd-proxy)](https://github.com/amaanx86/oci-prometheus-sd-proxy/releases)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/amaanx86/oci-prometheus-sd-proxy)](https://goreportcard.com/report/github.com/amaanx86/oci-prometheus-sd-proxy)
[![Docker Image](https://img.shields.io/badge/Docker-ghcr.io-blue?logo=docker)](https://github.com/amaanx86/oci-prometheus-sd-proxy/pkgs/container/oci-prometheus-sd-proxy)
[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/12388/badge)](https://www.bestpractices.dev/projects/12388)

Quick setup for running oci-prometheus-sd-proxy with Docker Compose.

## Quick Start

### 1. Prepare Configuration

```bash
# Copy example files
cp .env.example .env
cp config.yaml.example config.yaml

# Generate secure token
openssl rand -hex 32
```

Edit `.env` and set `SERVER_TOKEN` to the generated value.

Edit `config.yaml` and add your OCI tenancies (see comments in file).

### 2. Set Up OCI Keys

```bash
# Copy your OCI API keys to this directory
mkdir -p oci-keys
cp ~/.oci/api_key.pem oci-keys/
chmod 600 oci-keys/*
```

### 3. Start Service

```bash
# Start the container
docker-compose -f docker-compose-production.yml up -d

# View logs
docker-compose -f docker-compose-production.yml logs -f oci-sd-proxy

# Test API
TOKEN=$(grep SERVER_TOKEN .env | cut -d= -f2)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/targets
```

## Configuration

All settings are in `.env`:

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `SERVER_PORT` | 8080 | HTTP port |
| `SERVER_TOKEN` | - | **Required**: Bearer token for API auth |
| `DISCOVERY_TAG_KEY` | (empty) | OCI tag key to filter instances. Omit to discover all. |
| `DISCOVERY_TAG_VALUE` | (empty) | OCI tag value to filter instances. Omit to discover all. |
| `DISCOVERY_LINUX_PORT` | 9100 | node_exporter port |
| `DISCOVERY_WINDOWS_PORT` | 9182 | windows_exporter port |
| `DISCOVERY_REFRESH_INTERVAL` | 5m | Poll interval |
| `DISCOVERY_RATE_LIMIT_RPS` | 10.0 | OCI API requests per second per tenancy |

## File Structure

```text
deploy/docker/
├── docker-compose-production.yml  # Docker Compose config
├── .env.example                   # Environment template (copy to .env)
├── config.yaml.example            # OCI config template (copy to config.yaml)
├── oci-keys/                      # Place your OCI API keys here
├── config.yaml                    # Your OCI credentials (gitignored)
└── README.md                      # This file
```

## Usage

### Start

```bash
docker-compose -f docker-compose-production.yml up -d
```

### Stop

```bash
docker-compose -f docker-compose-production.yml down
```

### View Logs

```bash
docker-compose -f docker-compose-production.yml logs -f
```

### Health Check

```bash
curl http://localhost:8080/healthz
```

### Get Targets

```bash
TOKEN=$(grep SERVER_TOKEN .env | cut -d= -f2)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/targets | jq .
```

## Troubleshooting

**Image pull failed**: Ensure you have internet access or use a different image source.

**Config not found**: Verify `config.yaml` exists in current directory.

**No targets returned**: If tag filtering is configured, check that instances have the matching OCI tag. If tag filtering is disabled (default), verify the proxy has IAM permissions to list instances in the target compartments.

**Auth failed**: Verify `SERVER_TOKEN` in `.env` matches what's in `Authorization` header.
