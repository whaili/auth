{{/*
Expand the name of the chart.
*/}}
{{- define "bearer-token-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "bearer-token-service.fullname" -}}
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
{{- define "bearer-token-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "bearer-token-service.labels" -}}
helm.sh/chart: {{ include "bearer-token-service.chart" . }}
{{ include "bearer-token-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "bearer-token-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "bearer-token-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "bearer-token-service.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "bearer-token-service.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
MongoDB connection URI
*/}}
{{- define "bearer-token-service.mongoUri" -}}
{{- if .Values.mongodb.enabled }}
{{- printf "mongodb://%s:%s@mongodb:27017/%s?authSource=admin" .Values.mongodb.auth.username .Values.mongodb.auth.password .Values.mongodb.auth.database }}
{{- else }}
{{- .Values.externalMongodb.uri }}
{{- end }}
{{- end }}

{{/*
Redis address (host:port format)
*/}}
{{- define "bearer-token-service.redisAddr" -}}
{{- if .Values.redis.enabled -}}
redis:6379
{{- else -}}
{{- .Values.externalRedis.addr -}}
{{- end -}}
{{- end }}
