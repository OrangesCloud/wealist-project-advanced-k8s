{{/*
Standard deployment template for weAlist services
Usage in service chart:
  {{- include "wealist-common.deployment" . }}

Note: This template is only rendered when rollout.enabled is false (default).
When rollout.enabled is true, use the Argo Rollout template instead.
*/}}
{{- define "wealist-common.deployment" -}}
{{- if not (and .Values.rollout .Values.rollout.enabled) }}
{{/* Validate required values */}}
{{- include "wealist-common.validateRequired" (dict "values" .Values "required" (list "image.repository" "service.port" "service.targetPort")) }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "wealist-common.fullname" . }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  {{- $serviceName := include "wealist-common.name" . }}
  {{- $serviceReplicaCount := index .Values $serviceName "replicaCount" | default .Values.replicaCount | default 1 }}
  replicas: {{ $serviceReplicaCount }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "wealist-common.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "wealist-common.selectorLabels" . | nindent 8 }}
        {{/* Version label for DestinationRule subset matching - always required */}}
        {{- if and .Values.canary .Values.canary.enabled }}
        version: {{ .Values.canary.version | default "stable" }}
        {{- else }}
        version: stable
        {{- end }}
        {{/* Additional pod labels (e.g., stage: prod for Kyverno policy) */}}
        {{- if .Values.podLabels }}
        {{- toYaml .Values.podLabels | nindent 8 }}
        {{- end }}
      annotations:
        {{- if .Values.podAnnotations }}
        {{- toYaml .Values.podAnnotations | nindent 8 }}
        {{- end }}
        {{/* ConfigMap checksum - triggers pod restart on config change */}}
        checksum/config: {{ include "wealist-common.configChecksum" . }}
        {{/* Auto-add Prometheus annotations if metrics enabled */}}
        {{- if .Values.metrics }}
        {{- if .Values.metrics.enabled }}
        {{- include "wealist-common.prometheusAnnotations" (dict "port" .Values.service.targetPort "path" (.Values.metrics.path | default "/metrics")) | nindent 8 }}
        {{- end }}
        {{- end }}
        {{/* Istio Sidecar injection and resource configuration */}}
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
      {{- /* Init Container: Secret이 준비될 때까지 대기 */}}
      {{- if .Values.waitForSecrets }}
      {{- if .Values.waitForSecrets.enabled }}
      initContainers:
        - name: wait-for-secrets
          image: {{ .Values.waitForSecrets.image | default "bitnami/kubectl:1.30" }}
          imagePullPolicy: IfNotPresent
          command:
            - /bin/sh
            - -c
            - |
              echo "Waiting for secret {{ .Values.waitForSecrets.secretName }}..."
              TIMEOUT={{ .Values.waitForSecrets.timeout | default 300 }}
              ELAPSED=0
              while [ $ELAPSED -lt $TIMEOUT ]; do
                if kubectl get secret {{ .Values.waitForSecrets.secretName }} -n {{ .Release.Namespace }} -o jsonpath='{.data.DB_HOST}' 2>/dev/null | base64 -d | grep -q .; then
                  echo "Secret {{ .Values.waitForSecrets.secretName }} is ready!"
                  exit 0
                fi
                echo "Waiting for secret... ($ELAPSED/$TIMEOUT seconds)"
                sleep 5
                ELAPSED=$((ELAPSED + 5))
              done
              echo "Timeout waiting for secret {{ .Values.waitForSecrets.secretName }}!"
              exit 1
          securityContext:
            runAsNonRoot: true
            runAsUser: 1000
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
      {{- end }}
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
            {{- /* Include shared secret from wealist-infrastructure */}}
            - secretRef:
                name: wealist-shared-secret
            {{- /* Include ArgoCD secret (SealedSecret) - optional for fallback */}}
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
{{- end }}{{/* end if not rollout.enabled */}}
{{- end }}
