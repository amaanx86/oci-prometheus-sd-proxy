# oci-prometheus-sd-proxy Helm chart (MVP)

Helm chart for [oci-prometheus-sd-proxy](https://github.com/amaanx86/oci-prometheus-sd-proxy): a Prometheus HTTP Service Discovery proxy for Oracle Cloud Infrastructure compute instances.

This is an **MVP chart**. It deploys the proxy only. Operators supply OCI configuration and Secrets out-of-band.

## What this chart installs

- Deployment, Service (ClusterIP), ServiceAccount, and ConfigMap for `oci-prometheus-sd-proxy`
- Non-sensitive `config.yaml` rendered from `.Values.config`
- References to **existing** Kubernetes Secrets for `SERVER_TOKEN` and the OCI API private key
- Basic pod/container `securityContext` and HTTP liveness/readiness probes

## What this chart does not install or manage

- Prometheus (or any scrape configuration inside Prometheus)
- `node_exporter`, `windows_exporter`, or other instance agents
- Kubernetes Secrets containing tokens or private keys
- OCI IAM policies, dynamic groups, or network/firewall rules
- [ServiceMonitor](https://github.com/prometheus-operator/prometheus-operator) (intentionally omitted from this MVP)
- [ScrapeConfig](https://prometheus-operator.dev/docs/developer/scrapeconfig/) (possible later integration; out of scope here)

## Prerequisites

- Kubernetes cluster with Helm 3
- Network egress from the pod to OCI APIs for configured regions
- OCI IAM permissions for the configured tenancy user (API key auth) or instance principal setup if you use that auth mode
- Two **existing** Secrets (see below) created before or after install—the pod will not run until they exist
- Replace placeholder OCIDs, regions, and fingerprints in `.Values.config` with your real non-secret OCI settings

## Required existing Secrets

The chart **does not** create Secrets. Configure names in `values.yaml` (`serverTokenSecret`, `ociPrivateKeySecret`).

### SERVER_TOKEN Secret

Used as the container env var `SERVER_TOKEN` (Bearer token for `GET /v1/targets`).

Default reference: Secret `oci-sd-server-token`, key `server-token`.

```bash
# Generate a token locally (example)
TOKEN="$(openssl rand -hex 32)"

kubectl create secret generic oci-sd-server-token \
  --namespace monitoring \
  --from-literal=server-token="${TOKEN}"
```

Store the token securely; configure the same value in Prometheus separately (see below).

### OCI private key Secret

Mounted as a file at `{ociPrivateKeyMountPath}/{ociPrivateKeySecret.key}` (default: `/etc/oci-sd/keys/api_key.pem`).

Default reference: Secret `oci-sd-oci-key`, key `api_key.pem`.

```bash
kubectl create secret generic oci-sd-oci-key \
  --namespace monitoring \
  --from-file=api_key.pem=/path/to/your/api_key.pem
```

**Important:** `config.tenancies[].private_key_path` must match the mounted file path. With defaults, use `/etc/oci-sd/keys/api_key.pem`.

## Install

Create Secrets in the target namespace, then install:

```bash
helm install oci-sd ./charts/oci-prometheus-sd-proxy \
  --namespace monitoring \
  --create-namespace
```

## Values override example

Sensitive values must **not** go in `values.yaml`. Override Secret names and non-sensitive OCI config only:

```yaml
serverTokenSecret:
  name: my-sd-token
  key: server-token

ociPrivateKeySecret:
  name: my-oci-key
  key: api_key.pem

config:
  server:
    port: 8080
  discovery:
    tag_key: monitoring
    tag_value: enabled
    linux_port: 9100
    windows_port: 9182
    refresh_interval: 5m
    rate_limit_rps: 10.0
  tenancies:
    - name: my-tenancy
      auth_type: api_key
      region: us-ashburn-1
      tenancy_id: ocid1.tenancy.oc1..example
      user_id: ocid1.user.oc1..example
      fingerprint: "00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff"
      private_key_path: /etc/oci-sd/keys/api_key.pem
      compartments: []
```

Apply with:

```bash
helm install oci-sd ./charts/oci-prometheus-sd-proxy -f my-values.yaml -n monitoring
```

## config.yaml and CONFIG_PATH

- `.Values.config` is rendered into a ConfigMap key `config.yaml`.
- The Deployment sets `CONFIG_PATH` (default `/etc/oci-sd/config.yaml`) and mounts that file from the ConfigMap.
- Do **not** put `server.token` or PEM key content in `.Values.config`.
- Tenancies are YAML-only in the application; there are no env vars for tenancy blocks.

## Health endpoints

| Path | Auth | Notes |
|------|------|-------|
| `/healthz` | none | Liveness probe |
| `/readyz` | none | Readiness probe; returns 200 today and does **not** check OCI discovery health |
| `/v1/targets` | Bearer `SERVER_TOKEN` | Prometheus HTTP SD JSON |

## Prometheus configuration (separate from this chart)

Point Prometheus at the in-cluster Service with `http_sd_configs`. Token management in Prometheus is **your** responsibility; this chart does not configure Prometheus.

Service DNS pattern: `<release-fullname>.<namespace>.svc:<port>` (default port `8080`).

```yaml
scrape_configs:
  - job_name: oci-compute
    http_sd_configs:
      - url: http://oci-sd-oci-prometheus-sd-proxy.monitoring.svc:8080/v1/targets
        authorization:
          type: Bearer
          credentials: <your-server-token>
        refresh_interval: 60s
    relabel_configs:
      - source_labels: [__meta_oci_instance_name]
        target_label: instance
```

Replace the URL host with your release fullname and namespace. Use a Prometheus Secret or external secret manager for `<your-server-token>`—not Helm values.

## Validation

```bash
helm lint charts/oci-prometheus-sd-proxy
helm template oci-sd charts/oci-prometheus-sd-proxy
```

## Image

Default image: `ghcr.io/amaanx86/oci-prometheus-sd-proxy:latest`. Pin a release tag or digest in production (`values.yaml` → `image.tag`).
