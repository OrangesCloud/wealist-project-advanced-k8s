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

{{/*
Role for init container to read secrets
Usage: {{- include "wealist-common.secretReaderRole" . }}
*/}}
{{- define "wealist-common.secretReaderRole" -}}
{{- if .Values.waitForSecrets }}
{{- if .Values.waitForSecrets.enabled }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "wealist-common.fullname" . }}-secret-reader
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["{{ .Values.waitForSecrets.secretName }}"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "wealist-common.fullname" . }}-secret-reader
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
subjects:
  - kind: ServiceAccount
    name: {{ include "wealist-common.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
roleRef:
  kind: Role
  name: {{ include "wealist-common.fullname" . }}-secret-reader
  apiGroup: rbac.authorization.k8s.io
{{- end }}
{{- end }}
{{- end }}
