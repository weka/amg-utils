{{/*
Expand the name of the chart.
*/}}
{{- define "amg-cw-chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "amg-cw-chart.fullname" -}}
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
{{- define "amg-cw-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "amg-cw-chart.labels" -}}
helm.sh/chart: {{ include "amg-cw-chart.chart" . }}
{{ include "amg-cw-chart.selectorLabels" . }}
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
{{- define "amg-cw-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "amg-cw-chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
NVIDIA FS DaemonSet name
*/}}
{{- define "amg-cw-chart.nvidiafs.name" -}}
{{- printf "%s-install-nvidiafs" (include "amg-cw-chart.fullname" .) }}
{{- end }}

{{/*
NVIDIA FS labels
*/}}
{{- define "amg-cw-chart.nvidiafs.labels" -}}
{{ include "amg-cw-chart.labels" . }}
app.kubernetes.io/component: nvidia-fs-installer
{{- end }}

{{/*
NVIDIA FS selector labels
*/}}
{{- define "amg-cw-chart.nvidiafs.selectorLabels" -}}
{{ include "amg-cw-chart.selectorLabels" . }}
name: install-nvidiafs
app: install-nvidiafs
{{- end }}

{{/*
Weka AMG DaemonSet name
*/}}
{{- define "amg-cw-chart.wekaamg.name" -}}
{{- printf "%s-weka-amg" (include "amg-cw-chart.fullname" .) }}
{{- end }}

{{/*
Weka AMG labels
*/}}
{{- define "amg-cw-chart.wekaamg.labels" -}}
{{ include "amg-cw-chart.labels" . }}
app.kubernetes.io/component: weka-amg
{{- end }}

{{/*
Weka AMG selector labels
*/}}
{{- define "amg-cw-chart.wekaamg.selectorLabels" -}}
{{ include "amg-cw-chart.selectorLabels" . }}
name: weka-amg
app: weka-amg
{{- end }}
