{{/*
Expand the name of the chart.
*/}}
{{- define "cert-manager-config.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "cert-manager-config.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "cert-manager-config.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cert-manager-config.labels" -}}
helm.sh/chart: {{ include "cert-manager-config.chart" . }}
{{ include "cert-manager-config.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cert-manager-config.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cert-manager-config.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ClusterIssuer name
*/}}
{{- define "cert-manager-config.issuerName" -}}
{{- .Values.certManager.issuer.name | default "letsencrypt-prod" }}
{{- end }}

{{/*
Route53 secret name
*/}}
{{- define "cert-manager-config.route53SecretName" -}}
{{- .Values.certManager.route53Secret.name | default "route53-credentials" }}
{{- end }}
