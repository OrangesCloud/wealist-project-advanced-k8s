{{/*
Standard HorizontalPodAutoscaler template
Usage: {{- include "wealist-common.hpa" . }}
*/}}
{{- define "wealist-common.hpa" -}}
{{- if .Values.autoscaling }}
{{- if .Values.autoscaling.enabled }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "wealist-common.fullname" . }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "wealist-common.fullname" . }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
    {{- if .Values.autoscaling.targetCPUUtilizationPercentage }}
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
    {{- end }}
    {{- if .Values.autoscaling.targetMemoryUtilizationPercentage }}
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: {{ .Values.autoscaling.targetMemoryUtilizationPercentage }}
    {{- end }}
  {{- with .Values.autoscaling.behavior }}
  behavior:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
{{- end }}
{{- end }}
