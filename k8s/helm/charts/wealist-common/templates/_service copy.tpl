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
  {{- with .Values.service.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
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


