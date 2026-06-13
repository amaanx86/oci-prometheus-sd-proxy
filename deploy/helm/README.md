# OCI SD Proxy - Helm Chart for K8S Deployment

[![Go Report Card](https://goreportcard.com/badge/github.com/amaanax86/oci-prometheus-sd-proxy)](https://goreportcard.com/report/github.com/amaanax86/oci-prometheus-sd-proxy)
[![GitHub Release](https://img.shields.io/github/v/release/amaanax86/oci-prometheus-sd-proxy)](https://github.com/amaanax86/oci-prometheus-sd-proxy/releases)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docker Image](https://img.shields.io/badge/Docker-ghcr.io-blue?logo=docker)](https://github.com/amaanax86/oci-prometheus-sd-proxy/pkgs/container/oci-prometheus-sd-proxy)

Helm chart for [oci-prometheus-sd-proxy](https://github.com/amaanax86/oci-prometheus-sd-proxy) - a Prometheus HTTP Service Discovery proxy for Oracle Cloud Infrastructure compute instances.

## Quick Start

### 1. Create Secrets

```bash
# Bearer token for /v1/targets
TOKEN="$(openssl rand -hex 32)"
kubectl create secret generic oci-sd-server-token \
  --namespace monitoring \
  --from-literal=server-token="${TOKEN}"

# OCI API private key (skip if using instance_principal auth)
kubectl create secret generic oci-sd-oci-key \
  --namespace monitoring \
  --from-file=api_key.pem=~/.oci/api_key.pem
```

### 2. Configure OCI Settings

Copy and edit `values.yaml`, filling in your real tenancy OCIDs, region, user, and fingerprint under `config.tenancies`. Do not put the private key or token here.

### 3. Install

```bash
helm install oci-sd ./deploy/helm \
  --namespace monitoring \
  --create-namespace \
  -f my-values.yaml
```

### 4. Verify

```bash
# Check pod is running
kubectl get pods -n monitoring -l app.kubernetes.io/name=oci-prometheus-sd-proxy

# Health check
kubectl port-forward -n monitoring svc/oci-sd-oci-prometheus-sd-proxy 8080:8080
curl http://localhost:8080/healthz

# Fetch targets
curl -H "Authorization: Bearer ${TOKEN}" http://localhost:8080/v1/targets | jq .
```

## Prometheus Configuration

### Option A - ScrapeConfig CRD (prometheus-operator >= 0.65, recommended)

Reuses the same `serverTokenSecret` as the proxy - no credential duplication. Labels must match your `PrometheusSpec.scrapeConfigSelector`.

```yaml
scrapeConfig:
  enabled: true
  labels:
    release: <your-prometheus-release-name>
```

### Option B - http_sd_configs (standalone Prometheus or older operator)

Point Prometheus at the in-cluster Service using `http_sd_configs`. Create the scrape config as a Secret and reference it via `additionalScrapeConfigsSecret`, or add it directly to `prometheus.yml`:

```yaml
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
    - source_labels: [__meta_oci_tenancy_name]
      target_label: tenancy
    - source_labels: [__meta_oci_region]
      target_label: region
```

## Configuration

| Key | Default | Description |
| --- | ------- | ----------- |
| `replicaCount` | `1` | Replicas. Two is the practical maximum - more replicas multiply OCI API calls against the same rate limit. |
| `image.tag` | `"latest"` | Image tag. Pin a semver in production. |
| `image.pullPolicy` | `IfNotPresent` | Pull policy |
| `nameOverride` | `""` | Override chart name in resource names |
| `fullnameOverride` | `""` | Override full resource name |
| `imagePullSecrets` | `[]` | Pull secrets for private registries |
| `serviceAccount.create` | `true` | Create a ServiceAccount |
| `serviceAccount.annotations` | `{}` | SA annotations (e.g. for IRSA/Workload Identity) |
| `podAnnotations` | `{}` | Pod annotations |
| `podLabels` | `{}` | Extra pod labels |
| `service.type` | `ClusterIP` | Service type |
| `service.port` | `8080` | Service port |
| `service.annotations` | `{}` | Service annotations |
| `containerPort` | `8080` | Container listen port; keep in sync with `config.server.port` |
| `configPath` | `/etc/oci-sd/config.yaml` | Mount path for config; passed as `CONFIG_PATH` |
| `serverTokenSecret.name` | `oci-sd-server-token` | Secret name for `SERVER_TOKEN` |
| `serverTokenSecret.key` | `server-token` | Secret key for `SERVER_TOKEN` |
| `ociPrivateKeySecret.enabled` | `true` | Mount OCI private key. Set `false` for `instance_principal` auth. |
| `ociPrivateKeySecret.name` | `oci-sd-oci-key` | Secret name for OCI private key PEM |
| `ociPrivateKeySecret.key` | `api_key.pem` | Secret key for OCI private key PEM |
| `ociPrivateKeyMountPath` | `/etc/oci-sd/keys` | Directory where the key file is mounted |
| `resources` | `{}` | CPU/memory requests and limits |
| `livenessProbe.enabled` | `true` | Liveness probe on `/healthz` |
| `readinessProbe.enabled` | `true` | Readiness probe on `/readyz` |
| `updateStrategy` | `{}` | Deployment update strategy |
| `revisionHistoryLimit` | `3` | Old ReplicaSets to retain |
| `podDisruptionBudget.enabled` | `false` | Create a PodDisruptionBudget |
| `podDisruptionBudget.minAvailable` | `1` | Minimum available pods during voluntary disruption |
| `nodeSelector` | `{}` | Node selector |
| `tolerations` | `[]` | Tolerations |
| `affinity` | `{}` | Affinity rules |
| `topologySpreadConstraints` | `[]` | Topology spread constraints |
| `priorityClassName` | `""` | PriorityClass name |
| `extraEnv` | `[]` | Extra env vars (e.g. `HTTP_PROXY`) |
| `extraVolumes` | `[]` | Extra volumes (e.g. additional OCI key secrets) |
| `extraVolumeMounts` | `[]` | Extra volume mounts |
| `config` | see values.yaml | Non-sensitive `config.yaml` rendered into a ConfigMap |
| `scrapeConfig.enabled` | `false` | Create a `ScrapeConfig` CRD (prometheus-operator >= 0.65) |
| `scrapeConfig.labels` | `{}` | Labels added to the ScrapeConfig - must match `PrometheusSpec.scrapeConfigSelector` |
| `scrapeConfig.jobName` | `oci-compute` | Prometheus job name |
| `scrapeConfig.refreshInterval` | `60s` | Target list refresh interval |
| `scrapeConfig.relabelings` | see values.yaml | Relabeling rules for `__meta_oci_*` labels |
| `scrapeConfig.metricRelabelings` | `[]` | Metric relabeling rules |

## File Structure

```text
deploy/helm/
├── Chart.yaml
├── values.yaml
├── README.md
└── templates/
    ├── _helpers.tpl
    ├── configmap.yaml
    ├── deployment.yaml
    ├── service.yaml
    ├── serviceaccount.yaml
    ├── pdb.yaml          # PodDisruptionBudget (disabled by default)
    ├── scrapeconfig.yaml # ScrapeConfig CRD (disabled by default)
    └── NOTES.txt
```

## Health Endpoints

| Path | Auth | Notes |
| ---- | ---- | ----- |
| `/healthz` | none | Liveness probe |
| `/readyz` | none | Readiness probe |
| `/v1/targets` | Bearer `SERVER_TOKEN` | Prometheus HTTP SD endpoint |

## Validation

```bash
helm lint deploy/helm
helm template oci-sd deploy/helm --namespace monitoring
```

## Troubleshooting

**Pod stuck in Pending**: The Secrets referenced by `serverTokenSecret` and `ociPrivateKeySecret` must exist before the pod starts. Create them first or disable `ociPrivateKeySecret.enabled` if using instance_principal auth.

**No targets returned**: Check that instances have the configured OCI tag (`config.discovery.tag_key` / `tag_value`). Omit both to discover all running instances. Verify the pod has network egress to OCI API endpoints for the configured regions.

**Auth failed on /v1/targets**: The `SERVER_TOKEN` in the Secret must match the `Authorization: Bearer <token>` header sent by Prometheus.

**ScrapeConfig not picked up by Prometheus**: The labels on the ScrapeConfig must match `PrometheusSpec.scrapeConfigSelector`. Check with `kubectl get scrapeconfig -n monitoring`.

**Config changes not rolling out**: The Deployment uses a `checksum/config` annotation so it restarts automatically when the ConfigMap changes. If it doesn't, run `helm upgrade` to force a reconcile.
