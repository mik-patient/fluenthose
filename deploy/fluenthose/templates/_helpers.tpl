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
{{- if .Values.config.fluentbit_conf -}}
{{ .Values.config.fluentbit_conf }}
{{- else -}}
[SERVICE]
    HTTP_Server  On
    HTTP_Listen  0.0.0.0
    HTTP_PORT    2020
    Health_Check On 
    HC_Errors_Count 5 
    HC_Retry_Failure_Count 5 
    HC_Period 5
    Flush 1
    Parsers_File /fluent-bit/etc/parsers.conf

[INPUT]
    Name              forward
    Listen            127.0.0.1
    Port              24224
    Buffer_Chunk_Size 512K
    Buffer_Max_Size   512K

[FILTER]
    Name parser
    Match cloudfront
    Key_Name data
    Parser cloudfront
    Reserve_Data On

[FILTER]
    Name    lua
    Match   cloudfront
    script  /fluent-bit/etc/scripts.lua
    call    parseCloudfrontHeaders

[OUTPUT]
    name   loki
    match  *
    labels job=fluenthose, $type, 
    label_keys $csHost
    host {{ .Values.config.loki.address }}
    port {{ .Values.config.loki.port }}
    tls {{ .Values.config.loki.tls }}
    http_user {{ .Values.config.loki.auth.user }}
    http_passwd {{ .Values.config.loki.auth.password }}
{{- end }}
{{- end }}

{{/*

{{/* 
fluentbit parsers file
*/}}
{{- define "fluenthose.parsers.conf" -}}
[PARSER]
    Name cloudfront
    Format regex
    Regex ^(?<timestamp>[^\s]+)[\s]+(?<cIp>[^\s]+)[\s]+(?<timeToFirstByte>[^\s]+)[\s]+(?<scStatus>[^\s]+)[\s]+(?<scBytes>[^\s]+)[\s]+(?<csMethod>[^\s]+)[\s]+(?<csProtocol>[^\s]+)[\s]+(?<csHost>[^\s]+)[\s]+(?<csUriStem>[^\s]+)[\s]+(?<csBytes>[^\s]+)[\s]+(?<xEdgeLocation>[^\s]+)[\s]+(?<xEdgeRequestId>[^\s]+)[\s]+(?<xHostHeader>[^\s]+)[\s]+(?<timeTaken>[^\s]+)[\s]+(?<csProtocolVersion>[^\s]+)[\s]+(?<cIpVersion>[^\s]+)[\s]+(?<csUserAgent>[^\s]+)[\s]+(?<csReferer>[^\s]+)[\s]+(?<csCookie>[^\s]+)[\s]+(?<csUriQuery>[^\s]+)[\s]+(?<xEdgeResponseResultType>[^\s]+)[\s]+(?<xForwardedFor>[^\s]+)[\s]+(?<sslProtocol>[^\s]+)[\s]+(?<sslCipher>[^\s]+)[\s]+(?<xEdgeResultType>[^\s]+)[\s]+(?<fleEncryptedFields>[^\s]+)[\s]+(?<fleStatus>[^\s]+)[\s]+(?<scContentType>[^\s]+)[\s]+(?<scContentLen>[^\s]+)[\s]+(?<scRangeStart>[^\s]+)[\s]+(?<scRangeEnd>[^\s]+)[\s]+(?<cPort>[^\s]+)[\s]+(?<xEdgeDetailedResultType>[^\s]+)[\s]+(?<cCountry>[^\s]+)[\s]+(?<csAcceptEncoding>[^\s]+)[\s]+(?<csAccept>[^\s]+)[\s]+(?<cacheBehaviorPathPattern>[^\s]+)[\s]+(?<csHeaders>[^\s]+)[\s]+(?<csHeaderNames>[^\s]+)[\s]+(?<csHeadersCount>[^\s]+)$
    Time_Key timestamp
    Time_Format %s.%L
{{- end }}

{{/* 
fluentbit lua scripts file
*/}}
{{- define "fluenthose.scripts.lua" -}}
function parseCloudfrontHeaders(tag, timestamp, record)
    local rec_type = record["type"]
    if (rec_type == nil) then
        return 0, 0, 0
    end
    if (rec_type == "cloudfront") then
        local new_record = record
        local csHeaderNames = Unescape(record["csHeaderNames"])
        local csHeaders = Unescape(record["csHeaders"])
        local csCookie = Unescape(record["csCookie"])
        local csUserAgent = Unescape(record["csUserAgent"])

        new_record["csUserAgentParsed"] = csUserAgent
        new_record["csCookieParsed"] = CookieParser(csCookie)
        new_record["csHeadersParsed"] = HeaderParser(csHeaders)
        new_record["csHeaderNamesParsed"] = csHeaderNames
        return 2, timestamp, new_record
    else
        return 0, 0, 0
    end
end

Hex_to_char = function(x)
    return string.char(tonumber(x, 16))
end

Unescape = function(urlEncoded)
    if (urlEncoded == nil) then
        return ""
    end
    local urlDecoded = urlEncoded:gsub("%%(%x%x)", Hex_to_char)
    return urlDecoded
    -- return urlDecoded:gsub("\n", " ")
end

HeaderParser = function (x)
    local c = SplitHeaders(x)
    return c
    
end

SplitHeaders = function (x)
    local result = {}
    for line, v in x:gmatch("[^\r\n]+") do
        local key, value = line:match("^([^:]+):%s*(.+)$")
        if key then
            result[key:lower()] = value
        end
    end
    return result
    
end

CookieParser = function (x)
    local c = SplitCookies(x)
    return c
end

SplitCookies = function (x)
    local result = {}
    for k, v in x:gmatch("([^;%s]+)=([^;%s]+)") do
        if (k ~=  nil) then
            result[k:lower()] = Unescape(v)
        end
    end
    return result
end
{{- end }}    