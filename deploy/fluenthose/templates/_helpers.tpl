{{/*
Expand the name of the chart.
*/}}
{{- define "fluenthose.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "fluenthose.fullname" -}}
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
{{- define "fluenthose.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "fluenthose.labels" -}}
helm.sh/chart: {{ include "fluenthose.chart" . }}
{{ include "fluenthose.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "fluenthose.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fluenthose.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "fluenthose.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "fluenthose.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* 
fluentbit config file
*/}}
{{- define "fluenthose.fluentbit.conf" -}}
[SERVICE]
    HTTP_Server  On
    HTTP_Listen  0.0.0.0
    HTTP_PORT    2020
    Health_Check On 
    HC_Errors_Count 5 
    HC_Retry_Failure_Count 5 
    HC_Period 5

[INPUT]
    Name              forward
    Listen            127.0.0.1
    Port              24224
    Buffer_Chunk_Size 1M
    Buffer_Max_Size   6M
[OUTPUT]
    Name   stdout
    Match  *
{{- end }}
