package core

// dingtalk要有<br> 才可以换行
const dingAlertTemplate = `
{{- range .Alerts }}
{{- if eq .Status "firing" }}
<font color=#ff0000>======监控告警======</font><br>
告警时间: {{ BuildTime .StartsAt }}<br>
{{- else if eq .Status "resolved" }}
<font color=#00ff00>======告警恢复======</font><br>
恢复时间: {{ BuildTime .EndsAt }}<br>
{{- end }}
告警状态: {{ .Status }}<br>
告警类型: {{ .Labels.Alertname }}<br>
告警主题: {{ .Annotations.Summary }}<br>
告警详情: {{ .Annotations.Description }} {{ .Annotations.Message }}<br>
告警级别: {{ .Labels.Severity }}<br>
故障主机: {{ .Labels.Instance }} {{ .Labels.Pod }}<br>
处理人员: @18676730649
{{- end }}
`
const wechatAlertTemplate = `
{{- range .Alerts }}
{{- if eq .Status "firing" }}
<font color=#ff0000>======监控告警======</font>
告警时间: {{ BuildTime .StartsAt }}
{{- else if eq .Status "resolved" }}
<font color=#00ff00>======告警恢复======</font>
恢复时间: {{ BuildTime .EndsAt }}
{{- end }}
告警状态: {{ .Status }}
告警类型: {{ .Labels.Alertname }}
告警主题: {{ .Annotations.Summary }}
告警详情: {{ .Annotations.Description }} {{ .Annotations.Message }}
告警级别: {{ .Labels.Severity }}
故障主机: {{ .Labels.Instance }} {{ .Labels.Pod }}
<@祝禹>
{{- end }}

`
