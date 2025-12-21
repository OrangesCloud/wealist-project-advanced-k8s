{{/*
Standard service template for weAlist services
Usage in service chart:
  {{- include "wealist-common.service" . }}
*/}}
{{- define "wealist-common.service" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "wealist-common.fullname" . }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
  {{- if .Values.service.annotations }}
  annotations:
    {{- toYaml .Values.service.annotations | nindent 4 }}
  {{- end }}
spec:
  type: {{ .Values.service.type | default "ClusterIP" }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort }}
      protocol: TCP
      name: http
  selector:
    {{- include "wealist-common.selectorLabels" . | nindent 4 }}
{{- end }}

{{/*
Shared secret name
*/}}
{{- define "wealist-common.sharedSecretName" -}}
{{- .Values.global.sharedSecretName | default "wealist-shared-secret" }}
{{- end }}