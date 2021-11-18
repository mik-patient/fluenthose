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
    Parsers_File /fluent-bit/etc/parsers.conf

[INPUT]
    Name              forward
    Listen            127.0.0.1
    Port              24224
    Buffer_Chunk_Size 1M
    Buffer_Max_Size   6M

[FILTER]
    Name parser
    Match cloudfront
    Key_Name data
    Parser cloudfront
    Reserve_Data On

[OUTPUT]
    Name   stdout
    Match  *

[OUTPUT]
    name   loki
    match  *
    labels job=fluenthose, $type
    host {{ .Values.config.loki.address }}
    port {{ .Values.config.loki.port }}
    tls {{ .Values.config.loki.tls }}
    http_user {{ .Values.config.loki.auth.user }}
    http_passwd {{ .Values.config.loki.auth.password }}
{{- end }}

{{/* 
fluentbit parsers file
*/}}
{{- define "fluenthose.parsers.conf" -}}
[PARSER]
    Name cloudfront
    Format regex
    Regex ^(?<timestamp>[^\s]+)[\s]+(?<cIp>[^\s]+)[\s]+(?<timeToFirstByte>[^\s]+)[\s]+(?<scStatus>[^\s]+)[\s]+(?<scBytes>[^\s]+)[\s]+(?<csMethod>[^\s]+)[\s]+(?<csProtocol>[^\s]+)[\s]+(?<csHost>[^\s]+)[\s]+(?<csUriStem>[^\s]+)[\s]+(?<csBytes>[^\s]+)[\s]+(?<xEdgeLocation>[^\s]+)[\s]+(?<xEdgeRequestId>[^\s]+)[\s]+(?<xHostHeader>[^\s]+)[\s]+(?<timeTaken>[^\s]+)[\s]+(?<csProtocolVersion>[^\s]+)[\s]+(?<cIpVersion>[^\s]+)[\s]+(?<csUserAgent>[^\s]+)[\s]+(?<csReferer>[^\s]+)[\s]+(?<csCookie>[^\s]+)[\s]+(?<csUriQuery>[^\s]+)[\s]+(?<xEdgeResponseResultType>[^\s]+)[\s]+(?<xForwardedFor>[^\s]+)[\s]+(?<sslProtocol>[^\s]+)[\s]+(?<sslCipher>[^\s]+)[\s]+(?<xEdgeResultType>[^\s]+)[\s]+(?<fleEncryptedFields>[^\s]+)[\s]+(?<fleStatus>[^\s]+)[\s]+(?<scContentType>[^\s]+)[\s]+(?<scContentLen>[^\s]+)[\s]+(?<scRangeStart>[^\s]+)[\s]+(?<scRangeEnd>[^\s]+)[\s]+(?<cPort>[^\s]+)[\s]+(?<xEdgeDetailedResultType>[^\s]+)[\s]+(?<cCountry>[^\s]+)[\s]+(?<csAcceptEncoding>[^\s]+)[\s]+(?<csAccept>[^\s]+)[\s]+(?<cacheBehaviorPathPattern>[^\s]+)[\s]+(?<csHeaders>[^\s]+)[\s]+(?<csHeaderNames>[^\s]+)[\s]+(?<csHeadersCount>[^\s]+)$
{{- end }}