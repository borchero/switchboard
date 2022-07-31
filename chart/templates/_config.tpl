{{ define "config" }}
{{- $cfg := .Values.config -}}
apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
health:
  healthProbeBindAddress: :8081
leaderElection:
  leaderElect: {{ gt .Values.replicas 1.0 }}
  resourceName: {{ .Release.Name }}.switchboard.borchero.com
  resourceNamespace: {{ .Release.Namespace }}

{{ if .Values.metrics.enabled }}
metrics:
  bindAddress: :{{ .Values.metrics.port }}
{{ end }}

{{ if .Values.selector.ingressClass }}
selector:
  ingressClass: {{ .Values.selector.ingressClass }}
{{ end }}

{{- $certManager := .Values.integrations.certManager -}}
{{- $externalDNS := .Values.integrations.externalDNS -}}
{{ if or $certManager.enabled $externalDNS.enabled }}
integrations:
  {{ if $certManager.enabled }}
  certManager:
    {{ if $certManager.certificateTemplate }}
    certificateTemplate:
      {{ toYaml $certManager.certificateTemplate | nindent 6 }}
    {{ else if .Values.certificateIssuer.create }}
    certificateTemplate:
      spec:
        issuerRef:
          kind: ClusterIssuer
          name: {{ .Release.Name }}-letsencrypt-issuer
    {{ else }}
      {{ fail "certificate template is not provided and no issuer is created by this chart" }}
    {{ end }}
  {{ end }}
  {{ if $externalDNS.enabled }}
  externalDNS:
    {{ if and $externalDNS.targetService.name $externalDNS.targetService.namespace }}
    targetService:
      name: {{ $externalDNS.targetService.name }}
      namespace: {{ $externalDNS.targetService.namespace }}
    {{ else if $externalDNS.targetIPs }}
    targetIPs:
      {{ toYaml $externalDNS.targetIPs | nindent 6 }}
    {{ else }}
      {{ fail "exactly one of target service and target IPs must be set for external dns" }}
    {{ end }}
  {{ end }}
{{ end }}

{{ end }}