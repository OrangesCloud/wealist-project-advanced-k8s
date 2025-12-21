{{/*
Expand the name of the chart.
*/}}
{{- define "istio-config.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "istio-config.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "istio-config.labels" -}}
helm.sh/chart: {{ include "istio-config.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
