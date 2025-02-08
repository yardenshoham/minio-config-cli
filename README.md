# minio-config-cli

minio-config-cli is a MinIO utility to ensure the desired configuration state
for a server based on a JSON/YAML file. Store and handle the configuration files
inside git just like normal code. A MinIO restart isn't required to apply the
configuration.

Inspired by
[keycloak-config-cli](https://github.com/yardenshoham/minio-config-cli).

# Usage

```bash
minio-config-cli import MINIO_URL ACCESS_KEY SECRET_KEY --import-file-location=CONFIG_FILE1 --import-file-location=CONFIG_FILE2
```

Assuming you have a MinIO server running on `http://localhost:9000` with an
admin access key of `minioadmin`, a secret key of `minioadmin`, and a config
file at `/tmp/config.yaml`, you can import the config file with the following
command:

```bash
minio-config-cli import http://localhost:9000 minioadmin minioadmin --import-file-location=/tmp/config.yaml
```

# Config Files

Config files list resources to import into MinIO. An example config file is
shown below:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/yardenshoham/minio-config-cli/refs/heads/main/pkg/validation/schema.json
policies:
  - name: read-foobar-bucket
    policy: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "s3:GetObject"
            ],
            "Resource": [
              "arn:aws:s3:::foobar/*"
            ]
          }
        ]
      }
  - name: admin-reports-bucket
    policy: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "*"
            ],
            "Resource": [
              "arn:aws:s3:::admin-reports/*"
            ]
          }
        ]
      }
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
```

We provide a JSON schema file for correct creation of the config file. The
schema is available at [pkg/validation/schema.json](pkg/validation/schema.json).
As a URL:

```
https://raw.githubusercontent.com/yardenshoham/minio-config-cli/refs/heads/main/pkg/validation/schema.json
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
minio-config-cli import http://localhost:9000 minioadmin minioadmin \
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

### Docker run

```shell script
docker run \
    -v <your config path>:/config \
    yardenshoham/minio-config-cli:latest import http://host.docker.internal:9000 minioadmin minioadmin \
        --import-file-location=/config/*
```

### Docker build

You can build an own docker image by running

```shell
go build && docker build -t minio-config-cli .
```
