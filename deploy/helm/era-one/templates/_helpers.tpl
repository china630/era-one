{{- define "era-one.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "era-one.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "era-one.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
