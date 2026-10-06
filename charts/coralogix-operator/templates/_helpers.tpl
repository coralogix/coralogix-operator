{{/*
Expand the name of the chart.
*/}}
{{- define "coralogixOperator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "coralogixOperator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "coralogixOperator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "coralogixOperator.labels" -}}
helm.sh/chart: {{ include "coralogixOperator.chart" . }}
{{ include "coralogixOperator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.additionalLabels }}
{{ toYaml .Values.additionalLabels }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "coralogixOperator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "coralogixOperator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "coralogixOperator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "coralogixOperator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
Create the name of the service account to use
*/}}
{{- define "coralogixOperator.secretName" -}}
{{- if .Values.secret.create }}
{{- (include "coralogixOperator.fullname" .) }}
{{- else }}
{{- .Values.secret.secretKeyReference.name }}
{{- end }}
{{- end }}

{{/*
Get the name of the secret key, defaults to CORALOGIX_API_KEY
*/}}
{{- define "coralogixOperator.secretKey" -}}
{{- if and (.Values.secret.create) (not .Values.secret.secretKeyReference) }}
{{- print "CORALOGIX_API_KEY" }}
{{- else }}
{{- .Values.secret.secretKeyReference.key }}
{{- end }}
{{- end }}
