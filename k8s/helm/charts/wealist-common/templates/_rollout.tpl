{{/*
Argo Rollout template for weAlist services
Usage in service chart:
  {{- include "wealist-common.rollout" . }}

Requires values:
  rollout:
    enabled: true
    canary:
      steps:
        - setWeight: 10
        - pause: { duration: 5m }
        - setWeight: 30
        - pause: { duration: 5m }
        - setWeight: 50
        - pause: { duration: 5m }
*/}}
{{- define "wealist-common.rollout" -}}
{{- if .Values.rollout }}
{{- if .Values.rollout.enabled }}
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: {{ include "wealist-common.fullname" . }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount | default 1 }}
  {{- end }}
  revisionHistoryLimit: {{ .Values.rollout.revisionHistoryLimit | default 3 }}
  selector:
    matchLabels:
      {{- include "wealist-common.selectorLabels" . | nindent 6 }}
  strategy:
    canary:
      {{/* Stable and Canary Services for traffic splitting */}}
      stableService: {{ include "wealist-common.fullname" . }}
      canaryService: {{ include "wealist-common.fullname" . }}-canary
      {{/* Istio Traffic Routing */}}
      trafficRouting:
        istio:
          virtualServices:
          - name: wealist-routes
            routes:
            - primary
          destinationRule:
            name: {{ include "wealist-common.fullname" . }}
            stableSubsetName: stable
            canarySubsetName: canary
      {{/* Canary Steps */}}
      steps:
      {{- if .Values.rollout.canary }}
      {{- if .Values.rollout.canary.steps }}
      {{- toYaml .Values.rollout.canary.steps | nindent 6 }}
      {{- else }}
      {{/* Default canary steps */}}
      - setWeight: 10
      - pause: { duration: 2m }
      - setWeight: 30
      - pause: { duration: 2m }
      - setWeight: 50
      - pause: { duration: 2m }
      {{- end }}
      {{- else }}
      {{/* Default canary steps */}}
      - setWeight: 10
      - pause: { duration: 2m }
      - setWeight: 30
      - pause: { duration: 2m }
      - setWeight: 50
      - pause: { duration: 2m }
      {{- end }}
      {{/* Analysis Template for automated rollback */}}
      {{- if .Values.rollout.analysis }}
      {{- if .Values.rollout.analysis.enabled }}
      analysis:
        templates:
        - templateName: {{ include "wealist-common.fullname" . }}-analysis
        startingStep: {{ .Values.rollout.analysis.startingStep | default 1 }}
        args:
        - name: service-name
          value: {{ include "wealist-common.fullname" . }}
      {{- end }}
      {{- end }}
      {{/* Max Unavailable and Max Surge */}}
      maxUnavailable: {{ .Values.rollout.maxUnavailable | default 0 }}
      maxSurge: {{ .Values.rollout.maxSurge | default "25%" }}
  template:
    metadata:
      labels:
        {{- include "wealist-common.selectorLabels" . | nindent 8 }}
      annotations:
        {{- if .Values.podAnnotations }}
        {{- toYaml .Values.podAnnotations | nindent 8 }}
        {{- end }}
        {{- if .Values.metrics }}
        {{- if .Values.metrics.enabled }}
        {{- include "wealist-common.prometheusAnnotations" (dict "port" .Values.service.targetPort "path" (.Values.metrics.path | default "/metrics")) | nindent 8 }}
        {{- end }}
        {{- end }}
        checksum/config: {{ include "wealist-common.configChecksum" . }}
        {{/* Istio Sidecar injection */}}
        {{- if .Values.istio }}
        {{- if and .Values.istio.sidecar .Values.istio.sidecar.enabled }}
        sidecar.istio.io/inject: "true"
        {{- if .Values.istio.sidecar.resources }}
        {{- if .Values.istio.sidecar.resources.requests }}
        sidecar.istio.io/proxyCPU: {{ .Values.istio.sidecar.resources.requests.cpu | default "100m" | quote }}
        sidecar.istio.io/proxyMemory: {{ .Values.istio.sidecar.resources.requests.memory | default "128Mi" | quote }}
        {{- end }}
        {{- if .Values.istio.sidecar.resources.limits }}
        sidecar.istio.io/proxyCPULimit: {{ .Values.istio.sidecar.resources.limits.cpu | default "500m" | quote }}
        sidecar.istio.io/proxyMemoryLimit: {{ .Values.istio.sidecar.resources.limits.memory | default "256Mi" | quote }}
        {{- end }}
        {{- end }}
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
                name: wealist-shared-config
            - configMapRef:
                name: {{ include "wealist-common.fullname" . }}-config
            - secretRef:
                name: wealist-shared-secret
            - secretRef:
                name: wealist-argocd-secret
                optional: true
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
{{- end }}
{{- end }}

{{/*
Canary Service for Argo Rollouts
This service routes traffic to canary pods during progressive delivery
*/}}
{{- define "wealist-common.canaryService" -}}
{{- if .Values.rollout }}
{{- if .Values.rollout.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "wealist-common.fullname" . }}-canary
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
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
{{- end }}
{{- end }}

{{/*
AnalysisTemplate for automated rollback based on metrics
*/}}
{{- define "wealist-common.analysisTemplate" -}}
{{- if .Values.rollout }}
{{- if .Values.rollout.enabled }}
{{- if .Values.rollout.analysis }}
{{- if .Values.rollout.analysis.enabled }}
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: {{ include "wealist-common.fullname" . }}-analysis
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
spec:
  args:
  - name: service-name
  metrics:
  - name: success-rate
    interval: 30s
    count: {{ .Values.rollout.analysis.count | default 5 }}
    successCondition: result[0] >= {{ .Values.rollout.analysis.successThreshold | default 0.95 }}
    failureLimit: {{ .Values.rollout.analysis.failureLimit | default 3 }}
    provider:
      prometheus:
        address: {{ .Values.rollout.analysis.prometheusAddress | default "http://prometheus:9090" }}
        query: |
          sum(rate(
            istio_requests_total{
              destination_service_name="{{`{{args.service-name}}`}}",
              response_code!~"5.*"
            }[2m]
          )) / sum(rate(
            istio_requests_total{
              destination_service_name="{{`{{args.service-name}}`}}"
            }[2m]
          ))
  - name: latency-p99
    interval: 30s
    count: {{ .Values.rollout.analysis.count | default 5 }}
    successCondition: result[0] <= {{ .Values.rollout.analysis.latencyThresholdMs | default 500 }}
    failureLimit: {{ .Values.rollout.analysis.failureLimit | default 3 }}
    provider:
      prometheus:
        address: {{ .Values.rollout.analysis.prometheusAddress | default "http://prometheus:9090" }}
        query: |
          histogram_quantile(0.99,
            sum(rate(
              istio_request_duration_milliseconds_bucket{
                destination_service_name="{{`{{args.service-name}}`}}"
              }[2m]
            )) by (le)
          )
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
