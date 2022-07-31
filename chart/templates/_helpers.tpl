{{ define "image.tag" }}
{{- if .Values.image.tag -}}
{{ .Values.image.tag }}
{{- else -}}
{{ .Chart.Version }}
{{- end -}}
{{ end }}
