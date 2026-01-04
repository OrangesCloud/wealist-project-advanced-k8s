{{- define "istio-config.labels" -}}
app.kubernetes.io/name: istio-config
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "istio-config.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "istio-config.namespace" -}}
{{- .Values.global.namespace | default .Release.Namespace }}
{{- end }}
