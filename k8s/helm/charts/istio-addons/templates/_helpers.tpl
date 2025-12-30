{{- define "istio-addons.labels" -}}
app.kubernetes.io/name: istio-addons
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "istio-addons.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}
