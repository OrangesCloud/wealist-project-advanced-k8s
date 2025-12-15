{{/*
Standard deployment template for weAlist services
Usage in service chart:
  {{- include "wealist-common.deployment" . }}
*/}}
{{- define "wealist-common.deployment" -}}
{{/* Validate required values */}}
{{- include "wealist-common.validateRequired" (dict "values" .Values "required" (list "image.repository" "service.port" "service.targetPort")) }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "wealist-common.fullname" . }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling }}
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount | default 1 }}
  {{- end }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "wealist-common.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "wealist-common.selectorLabels" . | nindent 8 }}
      annotations:
        {{- if .Values.podAnnotations }}
        {{- toYaml .Values.podAnnotations | nindent 8 }}
        {{- end }}
        {{/* Auto-add Prometheus annotations if metrics enabled */}}
        {{- if .Values.metrics }}
        {{- if .Values.metrics.enabled }}
        {{- include "wealist-common.prometheusAnnotations" (dict "port" .Values.service.targetPort "path" (.Values.metrics.path | default "/metrics")) | nindent 8 }}
        {{- end }}
        {{- end }}
    spec:
      {{- if .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml .Values.imagePullSecrets | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "wealist-common.serviceAccountName" . }}
      {{- if .Values.podSecurityContext }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      {{- end }}
      containers:
        - name: {{ .Chart.Name }}
          image: {{ include "wealist-common.image" . | quote }}
          imagePullPolicy: {{ .Values.image.pullPolicy | default "Always" }}
          {{- if .Values.securityContext }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          {{- end }}
          ports:
            - name: http
              containerPort: {{ .Values.service.targetPort }}
              protocol: TCP
          {{- if .Values.healthCheck }}
          {{- if .Values.healthCheck.liveness }}
          {{- if ne (toString .Values.healthCheck.liveness.enabled) "false" }}
          livenessProbe:
            httpGet:
              path: {{ .Values.healthCheck.liveness.path | default "/health/live" }}
              port: {{ .Values.healthCheck.liveness.port | default .Values.service.targetPort }}
            initialDelaySeconds: {{ .Values.healthCheck.liveness.initialDelaySeconds | default 10 }}
            periodSeconds: {{ .Values.healthCheck.liveness.periodSeconds | default 10 }}
            timeoutSeconds: {{ .Values.healthCheck.liveness.timeoutSeconds | default 1 }}
            successThreshold: {{ .Values.healthCheck.liveness.successThreshold | default 1 }}
            failureThreshold: {{ .Values.healthCheck.liveness.failureThreshold | default 3 }}
          {{- end }}
          {{- end }}
          {{- if .Values.healthCheck.readiness }}
          {{- if ne (toString .Values.healthCheck.readiness.enabled) "false" }}
          readinessProbe:
            httpGet:
              path: {{ .Values.healthCheck.readiness.path | default "/health/ready" }}
              port: {{ .Values.healthCheck.readiness.port | default .Values.service.targetPort }}
            initialDelaySeconds: {{ .Values.healthCheck.readiness.initialDelaySeconds | default 5 }}
            periodSeconds: {{ .Values.healthCheck.readiness.periodSeconds | default 5 }}
            timeoutSeconds: {{ .Values.healthCheck.readiness.timeoutSeconds | default 1 }}
            successThreshold: {{ .Values.healthCheck.readiness.successThreshold | default 1 }}
            failureThreshold: {{ .Values.healthCheck.readiness.failureThreshold | default 3 }}
          {{- end }}
          {{- end }}
          {{- end }}
          {{- if .Values.resources }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- end }}
          envFrom:
            - configMapRef:
                name: {{ include "wealist-common.fullname" . }}-config
            {{- /* Always include shared secret - created by wealist-infrastructure */}}
            - secretRef:
                name: wealist-shared-secret
          {{- if .Values.envFrom }}
            {{- toYaml .Values.envFrom | nindent 12 }}
          {{- end }}
          {{- if .Values.env }}
          env:
            {{- toYaml .Values.env | nindent 12 }}
          {{- end }}
          {{- if .Values.volumeMounts }}
          volumeMounts:
            {{- toYaml .Values.volumeMounts | nindent 12 }}
          {{- end }}
      {{- if .Values.volumes }}
      volumes:
        {{- toYaml .Values.volumes | nindent 8 }}
      {{- end }}
      {{- if .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml .Values.nodeSelector | nindent 8 }}
      {{- end }}
      {{- if .Values.affinity }}
      affinity:
        {{- toYaml .Values.affinity | nindent 8 }}
      {{- end }}
      {{- if .Values.tolerations }}
      tolerations:
        {{- toYaml .Values.tolerations | nindent 8 }}
      {{- end }}
{{- end }}
