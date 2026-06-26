package app

const postgresLocalConfigTemplate = `{{ if .EnvFile -}}runtime:
  env_file: {{ .EnvFile }}

{{- end }}
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password_env: BACKUPCTL_POSTGRES_PASSWORD
    name: app
    sslmode: disable

backup:
  type: full
  compression: gzip

storage:
  type: local
  local:
    path: ./backups

logging:
  level: info
`

const mongoLocalConfigTemplate = `{{ if .EnvFile -}}runtime:
  env_file: {{ .EnvFile }}

{{- end }}
database:
  type: mongo
  mongo:
    uri_env: BACKUPCTL_MONGO_URI
    database: app

backup:
  type: full
  compression: gzip

storage:
  type: local
  local:
    path: ./backups

logging:
  level: info
`

const postgresS3ConfigTemplate = `{{ if .EnvFile -}}runtime:
  env_file: {{ .EnvFile }}

{{- end }}
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password_env: BACKUPCTL_POSTGRES_PASSWORD
    name: app
    sslmode: disable

backup:
  type: full
  compression: gzip

storage:
  type: s3
  s3:
    bucket: your-bucket
    region: us-east-1
    prefix: backupctl
    endpoint: ""
    force_path_style: false

logging:
  level: info
`

const mongoS3ConfigTemplate = `{{ if .EnvFile -}}runtime:
  env_file: {{ .EnvFile }}

{{- end }}
database:
  type: mongo
  mongo:
    uri_env: BACKUPCTL_MONGO_URI
    database: app

backup:
  type: full
  compression: gzip

storage:
  type: s3
  s3:
    bucket: your-bucket
    region: us-east-1
    prefix: backupctl
    endpoint: ""
    force_path_style: false

logging:
  level: info
`

const envTemplate = `BACKUPCTL_POSTGRES_PASSWORD=change-me
BACKUPCTL_MONGO_URI=mongodb://localhost:27017
BACKUPCTL_ENCRYPTION_PASSWORD=change-me
AWS_ACCESS_KEY_ID=change-me
AWS_SECRET_ACCESS_KEY=change-me
`
