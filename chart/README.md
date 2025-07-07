# minio-config-cli Helm Chart

minio-config-cli is a MinIO utility to ensure the desired configuration state
for a server based on a JSON/YAML file. Store and handle the configuration files
inside git just like normal code. A MinIO restart isn't required to apply the
configuration.

## TL;DR

```bash
helm install my-release oci://registry-1.docker.io/yardenshohamcharts/minio-config-cli
```

## Introduction

This chart bootstraps a [minio-config-cli](https://github.com/yardenshoham/minio-config-cli) job on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8.0+

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
helm install my-release oci://registry-1.docker.io/yardenshohamcharts/minio-config-cli
```

The command deploys minio-config-cli on the Kubernetes cluster in the default configuration. The [parameters](#parameters) section lists the parameters that can be configured during installation.

## Configuration and installation details

### Additional environment variables

In case you want to add extra environment variables, you can use the `extraEnvVars` property.

```yaml
extraEnvVars:
  - name: KEY
    value: value
```

Alternatively, you can use a ConfigMap or a Secret with the environment variables. To do so, use the `extraEnvVarsCM` or the `extraEnvVarsSecret` values.

### Sidecars

If additional init containers are needed in the same pod, they can be defined using the `initContainers` parameter. Here is an example:

```yaml
initContainers:
  - name: your-image-name
    image: your-image
    imagePullPolicy: Always
```

Learn more about [init containers](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/).

### Parameters

| Name                                                | Description                                                                                 | Value            |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------- | ---------------- |
| `global.imageRegistry`                              | Global Docker image registry                                                                | `""`             |
| `global.imagePullSecrets`                           | Global Docker registry secret names as an array                                             | `[]`             |
| `useHooks`                                          | Use Helm hooks to install the job                                                           | `false`          |
| `url`                                               | MinIO URL                                                                                   | REQUIRED         |
| `accessKey`                                         | MinIO access key                                                                            | REQUIRED         |
| `secretKey`                                         | MinIO secret key                                                                            | REQUIRED         |
| `config`                                            | minio-config-cli config file as a YAML/JSON object                                          | `{}`             |
| `extraConfig`                                       | Additional optional minio-config-cli config file as a YAML/JSON object                      | `{}`             |
| `nameOverride`                                      | String to partially override common.names.name                                              | `""`             |
| `fullnameOverride`                                  | String to fully override common.names.fullname                                              | `""`             |
| `commonAnnotations`                                 | Annotations to add to all deployed objects                                                  | `{}`             |
| `commonLabels`                                      | Labels to add to all deployed objects                                                       | `{}`             |
| `extraDeploy`                                       | Array of extra objects to deploy with the release                                           | `[]`             |
| `backoffLimit`                                      | Number of retries before considering a job as failed                                        | `10`             |
| `command`                                           | Override default container command (useful when using custom images)                        | `[]`             |
| `args`                                              | Override default container args (useful when using custom images)                           | `[]`             |
| `containerSecurityContext.enabled`                  | Enabled minio-config-cli containers' Security Context                                       | `true`           |
| `containerSecurityContext.seLinuxOptions`           | Set SELinux options in container                                                            | `{}`             |
| `containerSecurityContext.runAsUser`                | Set minio-config-cli containers' Security Context runAsUser                                 | `1001`           |
| `containerSecurityContext.runAsGroup`               | Set minio-config-cli containers' Security Context runAsGroup                                | `1001`           |
| `containerSecurityContext.runAsNonRoot`             | Set minio-config-cli containers' Security Context runAsNonRoot                              | `true`           |
| `containerSecurityContext.privileged`               | Set minio-config-cli containers' Security Context privileged                                | `false`          |
| `containerSecurityContext.readOnlyRootFilesystem`   | Set minio-config-cli containers' Security Context runAsNonRoot                              | `true`           |
| `containerSecurityContext.allowPrivilegeEscalation` | Set minio-config-cli container's privilege escalation                                       | `false`          |
| `containerSecurityContext.capabilities.drop`        | Set minio-config-cli container's Security Context runAsNonRoot                              | `["ALL"]`        |
| `containerSecurityContext.seccompProfile.type`      | Set minio-config-cli container's Security Context seccomp profile                           | `RuntimeDefault` |
| `podSecurityContext.enabled`                        | Enabled minio-config-cli pods' Security Context                                             | `true`           |
| `podSecurityContext.fsGroupChangePolicy`            | Set filesystem group change policy                                                          | `Always`         |
| `podSecurityContext.sysctls`                        | Set kernel settings using the sysctl interface                                              | `[]`             |
| `podSecurityContext.supplementalGroups`             | Set filesystem extra groups                                                                 | `[]`             |
| `podSecurityContext.fsGroup`                        | Set minio-config-cli pod's Security Context fsGroup                                         | `1001`           |
| `extraEnvVars`                                      | Array with extra environment variables to add to minio-config-cli pods                      | `[]`             |
| `extraEnvVarsCM`                                    | Name of existing ConfigMap containing extra env vars for minio-config-cli pods              | `""`             |
| `extraEnvVarsSecret`                                | Name of existing Secret containing extra env vars for minio-config-cli pods                 | `""`             |
| `extraVolumes`                                      | Optionally specify extra list of additional volumes for the minio-config-cli pods           | `[]`             |
| `extraVolumeMounts`                                 | Optionally specify extra list of additional volumeMounts for the minio-config-cli container | `[]`             |
| `initContainers`                                    | Add additional init containers to the minio-config-cli pods                                 | `[]`             |
| `resources`                                         | Set minio-config-cli pods' resource requests and limits                                     | `{}`             |
| `customLivenessProbe`                               | Custom livenessProbe that overrides the default one                                         | `{}`             |
| `customReadinessProbe`                              | Custom readinessProbe that overrides the default one                                        | `{}`             |
| `customStartupProbe`                                | Custom startupProbe that overrides the default one                                          | `{}`             |
| `automountServiceAccountToken`                      | Automount service account token for the server service account                              | `false`          |
| `hostAliases`                                       | minio-config-cli pods host aliases                                                          | `[]`             |
| `annotations`                                       | Additional custom annotations for minio-config-cli pods                                     | `{}`             |
| `podLabels`                                         | Extra labels for minio-config-cli pods                                                      | `{}`             |
| `podAnnotations`                                    | Annotations for minio-config-cli pods                                                       | `{}`             |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
helm install my-release \
  --set useHooks=true \
    oci://registry-1.docker.io/yardenshohamcharts/minio-config-cli
```

The above command instructs the Job to be a Helm hook.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```bash
helm install my-release -f values.yaml oci://registry-1.docker.io/yardenshohamcharts/minio-config-cli
```
