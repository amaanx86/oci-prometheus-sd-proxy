{{/*
Expand the name of the chart.
*/}}
{{- define "oci-prometheus-sd-proxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "oci-prometheus-sd-proxy.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label value (chart name-version).
*/}}
{{- define "oci-prometheus-sd-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to chart resources.
*/}}
{{- define "oci-prometheus-sd-proxy.labels" -}}
helm.sh/chart: {{ include "oci-prometheus-sd-proxy.chart" . }}
{{ include "oci-prometheus-sd-proxy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels (stable across upgrades; used by Service and Deployment).
*/}}
{{- define "oci-prometheus-sd-proxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "oci-prometheus-sd-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name for use in later Deployment templates.
*/}}
{{- define "oci-prometheus-sd-proxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "oci-prometheus-sd-proxy.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
