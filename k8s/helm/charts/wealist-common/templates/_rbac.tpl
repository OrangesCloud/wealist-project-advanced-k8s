{{/*
Standard ServiceAccount template
Usage: {{- include "wealist-common.serviceAccount" . }}
*/}}
{{- define "wealist-common.serviceAccount" -}}
{{- if .Values.serviceAccount }}
{{- if .Values.serviceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Values.serviceAccount.name | default (include "wealist-common.fullname" .) }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- if .Values.serviceAccount.imagePullSecrets }}
imagePullSecrets:
  {{- range .Values.serviceAccount.imagePullSecrets }}
  - name: {{ . }}
  {{- end }}
{{- else if .Values.global.imagePullSecrets }}
imagePullSecrets:
  {{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
  {{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
ServiceAccount name
*/}}
{{- define "wealist-common.serviceAccountName" -}}
{{- if .Values.serviceAccount }}
{{- if .Values.serviceAccount.create }}
{{- .Values.serviceAccount.name | default (include "wealist-common.fullname" .) }}
{{- else }}
{{- .Values.serviceAccount.name | default "default" }}
{{- end }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}
