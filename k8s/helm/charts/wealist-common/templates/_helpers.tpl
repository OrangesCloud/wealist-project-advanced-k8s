{{/*
Expand the name of the chart.
*/}}
{{- define "wealist-common.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "wealist-common.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "wealist-common.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "wealist-common.labels" -}}
helm.sh/chart: {{ include "wealist-common.chart" . }}
{{ include "wealist-common.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: wealist
{{- if .Values.global }}
{{- if .Values.global.environment }}
app.kubernetes.io/env: {{ .Values.global.environment }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "wealist-common.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wealist-common.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Database URL constructor
Usage: {{ include "wealist-common.databaseURL" (dict "user" "user_service" "password" "pass" "host" "postgres" "port" "5432" "db" "wealist_user_db") }}
*/}}
{{- define "wealist-common.databaseURL" -}}
postgresql://{{ .user }}:{{ .password }}@{{ .host }}:{{ .port }}/{{ .db }}?sslmode=disable
{{- end }}

{{/*
Image name constructor
Combines global registry with image repository and tag
*/}}
{{- define "wealist-common.image" -}}
{{- if .Values.global }}
{{- if .Values.global.imageRegistry }}
{{- printf "%s/%s:%s" .Values.global.imageRegistry .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- end }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- end }}
{{- end }}

{{/*
Validate required values
Usage: {{ include "wealist-common.validateRequired" (dict "values" .Values "required" (list "image.repository" "service.port")) }}
*/}}
{{- define "wealist-common.validateRequired" -}}
{{- $values := .values -}}
{{- $required := .required -}}
{{- range $required }}
  {{- $path := . }}
  {{- $parts := splitList "." $path }}
  {{- $current := $values }}
  {{- $found := true }}
  {{- range $parts }}
    {{- if hasKey $current . }}
      {{- $current = index $current . }}
    {{- else }}
      {{- $found = false }}
    {{- end }}
  {{- end }}
  {{- if not $found }}
    {{- fail (printf "Required value '%s' is not set" $path) }}
  {{- end }}
  {{/* Check if value is empty - only for string types */}}
  {{- if kindIs "string" $current }}
    {{- if eq $current "" }}
      {{- fail (printf "Required value '%s' cannot be empty" $path) }}
    {{- end }}
  {{- end }}
  {{/* Check if value is nil */}}
  {{- if kindIs "invalid" $current }}
    {{- fail (printf "Required value '%s' is nil" $path) }}
  {{- end }}
{{- end }}
{{- end }}

{{/*
Environment-specific configuration merger
Merges global and service-specific config
*/}}
{{- define "wealist-common.envConfig" -}}
{{- $global := .Values.global | default dict }}
{{- $local := .Values.config | default dict }}
{{- merge $local $global | toYaml }}
{{- end }}

{{/*
Standard security context
*/}}
{{- define "wealist-common.securityContext" -}}
runAsNonRoot: true
runAsUser: 1000
fsGroup: 1000
capabilities:
  drop:
    - ALL
readOnlyRootFilesystem: false
allowPrivilegeEscalation: false
{{- end }}

{{/*
Resource naming with environment prefix
Usage: {{ include "wealist-common.resourceName" (dict "name" "my-resource" "context" .) }}
*/}}
{{- define "wealist-common.resourceName" -}}
{{- $name := .name -}}
{{- $context := .context -}}
{{- if $context.Values.global }}
{{- if $context.Values.global.environment }}
{{- printf "%s-%s" $context.Values.global.environment $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- else }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Standard Prometheus annotations
*/}}
{{- define "wealist-common.prometheusAnnotations" -}}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .port | quote }}
prometheus.io/path: {{ .path | default "/metrics" | quote }}
{{- end }}
