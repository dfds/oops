
{{- define "oops.serviceaccount.awsRoleArn" -}}
{{- if .Values.serviceAccount.awsRoleArn }}
{{- .Values.serviceAccount.awsRoleArn }}
{{- else }}
{{- "arn:aws:iam::00000000000:role/replaceme" }}
{{- end }}
{{- end }}