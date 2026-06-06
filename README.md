# minio-config-cli

[![Go Report Card](https://goreportcard.com/badge/github.com/yardenshoham/minio-config-cli)](https://goreportcard.com/report/github.com/yardenshoham/minio-config-cli)

minio-config-cli is a MinIO utility to ensure the desired configuration state
for a server based on a JSON/YAML file. Store and handle the configuration files
inside git just like normal code. A MinIO restart isn't required to apply the
configuration.

Inspired by
[keycloak-config-cli](https://github.com/yardenshoham/minio-config-cli).

# Usage

The `import` subcommand takes the MinIO URL as a positional argument and
authenticates using either static access-key/secret-key credentials or OIDC
(AssumeRoleWithWebIdentity).

## Static credentials

Assuming you have a MinIO server running on `http://localhost:9000` with an
admin access key of `minioadmin`, a secret key of `minioadmin`, and a config
file at `/tmp/config.yaml`, you can import the config file with the following
command:

```bash
minio-config-cli import http://localhost:9000 \
    --access-key=minioadmin --secret-key=minioadmin \
    --import-file-location=/tmp/config.yaml
```

## OIDC credentials

```bash
minio-config-cli import https://minio.example.com \
    --oidc-issuer-url=https://keycloak.example.com/realms/minio \
    --oidc-client-id=minio-client \
    --oidc-client-secret=$OIDC_CLIENT_SECRET \
    --import-file-location=/tmp/config.yaml
```

For a password grant, pass `--username` and `--password`
instead of `--oidc-client-secret`. The grant is auto-detected from the
flags but can be forced with `--grant-type=client-credentials` or
`--grant-type=password`.

All flags fall back to environment variables when unset:
`MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`,
`OIDC_CLIENT_SECRET`, `OIDC_EXTRA_SCOPES` (comma-separated), `OIDC_GRANT_TYPE`,
`OIDC_USERNAME`, `OIDC_PASSWORD`.

# Config Files

Config files list resources to import into MinIO. An example config file is
shown below:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/yardenshoham/minio-config-cli/refs/heads/main/pkg/validation/schema.json
policies:
  - name: read-foobar-bucket
    policy:
      Version: "2012-10-17"
      Statement:
        - Effect: Allow
          Action: [s3:GetObject]
          Resource: [arn:aws:s3:::foobar/*]
  - name: admin-reports-bucket
    policy:
      Version: "2012-10-17"
      Statement:
        - Effect: Allow
          Action: ["*"]
          Resource: [arn:aws:s3:::admin-reports/*]
users:
  - accessKey: iamenabled
    secretKey: mypasswordisnice
    status: enabled
    policies: [read-foobar-bucket]
  - accessKey: iamdisabled
    secretKey: mypasswordisawesome
    status: disabled
  - accessKey: iamprettysureiamenabled
    secretKey: mypasswordisnicemypasswordisnicemypasswordisnice
buckets:
  - name: foobar
    lifecycle:
      Rules:
        - ID: rule1
          Status: Enabled
          Expiration:
            Days: 14
  - name: admin-reports
    quota:
      size: 10737418240 # 10Gi
  - name: static-assets-public
    policy:
      Version: "2012-10-17"
      Statement:
        - Effect: Allow
          Action:
            - s3:GetObject
            - s3:ListBucket
          Resource: "arn:aws:s3:::*"
          Principal:
            AWS:
              - "*"
  - name: versioned-bucket
    versioning:
      Status: Enabled # Enable bucket versioning
  - name: versioned-with-exclusions
    versioning:
      Status: Enabled
      ExcludedPrefixes: # Exclude specific prefixes from versioning
        - Prefix: "logs/"
        - Prefix: "tmp/"
      ExcludeFolders: true # Exclude folders from versioning
```

We provide a JSON schema file for correct creation of the config file. The
schema is available at [pkg/validation/schema.json](pkg/validation/schema.json).
As a URL:

```
https://raw.githubusercontent.com/yardenshoham/minio-config-cli/refs/heads/main/pkg/validation/schema.json
```

# Variable Substitution

Config files support variable substitution using the syntax `$(prefix:key)`.
Substitution is always enabled and is applied to the raw config text before
validation and unmarshaling.

## Syntax

```
$(prefix:key)
```

Expressions are resolved inside-out, so nesting is supported:

```
$(file:$(env:CONFIG_PATH))
```

To include a literal `$(prefix:key)` string without substitution, escape it with
a double `$`:

```
$$(env:HOME)  →  $(env:HOME)
```

## Supported Prefixes

| Prefix          | Description                                    | Example                                                  |
| --------------- | ---------------------------------------------- | -------------------------------------------------------- |
| `env`           | Value of an environment variable               | `$(env:HOME)` → `/home/user`                             |
| `file`          | Contents of a file (relative to working dir)   | `$(file:secrets/key.txt)` → file contents                |
| `base64Decoder` | Decode a Base64 string                         | `$(base64Decoder:SGVsbG8=)` → `Hello`                    |
| `base64Encoder` | Encode a string to Base64                      | `$(base64Encoder:Hello)` → `SGVsbG8=`                    |
| `urlDecoder`    | URL-decode a string                            | `$(urlDecoder:Hello%20World)` → `Hello World`            |
| `urlEncoder`    | URL-encode a string                            | `$(urlEncoder:Hello World)` → `Hello+World`              |
| `url`           | Fetch content from an HTTP, HTTPS, or file URL | `$(url:https://example.com/policy.json)` → response body |

## Example

```yaml
buckets:
  - name: $(env:BUCKET_NAME)
users:
  - accessKey: $(env:MINIO_ACCESS_KEY)
    secretKey: $(file:secrets/minio-secret-key.txt)
    status: enabled
policies:
  - name: my-policy
    policy: $(url:https://example.com/policy.json)
```

# Build this project

```bash
go build
```

# Run tests

We use `testcontainers` so we test against an actual MinIO server.

```bash
go test ./...
```

# Run this project

Run a local instance of the MinIO server on port 9000:

```bash
docker run --rm -p 9000:9000 -p 9001:9001 minio/minio server /data --console-address ":9001"
```

before performing the following command:

```bash
minio-config-cli import http://localhost:9000 \
    --access-key=minioadmin --secret-key=minioadmin \
    --import-file-location=./testdata/config.yaml
```

## Docker

Docker images are available at
[DockerHub](https://hub.docker.com/r/yardenshoham/minio-config-cli)
(docker.io/yardenshoham/minio-config-cli).

Available docker tags

| Tag      | Description                                   |
| -------- | --------------------------------------------- |
| `latest` | latest available release of minio-config-cli. |
| `va.b.c` | minio-config-cli version `a.b.c` .            |
| `a.b.c`  | minio-config-cli version `a.b.c` .            |

### Docker run

```shell script
docker run \
    -v <your config path>:/config \
    yardenshoham/minio-config-cli:latest import http://host.docker.internal:9000 \
        --access-key=minioadmin --secret-key=minioadmin \
        --import-file-location=/config/*
```

### Docker build

You can build an own docker image by running

```shell
CGO_ENABLED=0 go build && docker build -t minio-config-cli .
```

## Helm

We provide a helm chart [here](chart).

Since it makes no sense to deploy minio-config-cli as standalone application, you could add it as dependency to your chart deployment.

Checkout helm docs about [chart dependencies](https://helm.sh/docs/topics/charts/#chart-dependencies)!
