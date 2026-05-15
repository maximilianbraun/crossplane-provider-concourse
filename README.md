# crossplane-provider-concourse

A [Crossplane](https://crossplane.io) provider for managing [Concourse CI](https://concourse-ci.org) resources declaratively. Define your teams, pipelines, jobs, builds, and workers as Kubernetes managed resources — the provider reconciles them continuously against your Concourse server.

## Overview

Built on [crossplane-runtime v2](https://github.com/crossplane/crossplane-runtime) and the official [go-concourse](https://github.com/concourse/concourse) client library. Supports Concourse v7+.

**API groups:**
- `concourse.crossplane.io/v1alpha1` — ProviderConfig
- `ci.concourse.crossplane.io/v1alpha1` — managed resources

| Resource | Purpose |
|---|---|
| `ProviderConfig` | Connection + auth to a Concourse server |
| `Team` | Team management with role bindings |
| `Pipeline` | Pipeline configuration (inline or ConfigMap-sourced) |
| `Job` | Job pause/unpause control |
| `Build` | Trigger builds and abort running ones |
| `PipelineResource` | Resource version pinning |
| `Worker` | Worker lifecycle management (land/retire/prune) |

## Quick Start

### 1. Install the provider

```sh
kubectl crossplane install provider ghcr.io/maximilianbraun/crossplane-provider-concourse:latest
```

### 2. Create credentials and ProviderConfig

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: concourse-creds
  namespace: crossplane-system
stringData:
  password: your-password
---
apiVersion: concourse.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  url: https://concourse.example.com
  credentials:
    basicAuth:
      username: admin
      passwordSecretRef:
        name: concourse-creds
        namespace: crossplane-system
        key: password
```

Bearer token auth is also supported:

```yaml
spec:
  credentials:
    bearerTokenSecretRef:
      name: concourse-token
      namespace: crossplane-system
      key: token
```

Custom TLS (optional):

```yaml
spec:
  tls:
    insecureSkipVerify: false
    caSecretRef:
      name: concourse-ca
      namespace: crossplane-system
      key: ca.crt
```

### 3. Create a Team

```yaml
apiVersion: ci.concourse.crossplane.io/v1alpha1
kind: Team
metadata:
  name: my-team
spec:
  forProvider:
    teamName: my-team
    roles:
      - name: owner
        users:
          - admin
      - name: member
        groups:
          - github:my-org:developers
  providerConfigRef:
    name: default
```

### 4. Deploy a Pipeline

```yaml
apiVersion: ci.concourse.crossplane.io/v1alpha1
kind: Pipeline
metadata:
  name: hello-world
spec:
  forProvider:
    teamName: my-team
    pipelineName: hello-world
    paused: false
    exposed: false
    config:
      inline: |
        jobs:
          - name: hello
            plan:
              - task: say-hello
                config:
                  platform: linux
                  image_resource:
                    type: registry-image
                    source:
                      repository: alpine
                  run:
                    path: echo
                    args: ["Hello, Crossplane!"]
  providerConfigRef:
    name: default
```

ConfigMap-sourced config is also supported via `config.configMapRef`.

### 5. Trigger a Build

```yaml
apiVersion: ci.concourse.crossplane.io/v1alpha1
kind: Build
metadata:
  name: hello-build-1
spec:
  forProvider:
    teamName: my-team
    pipelineName: hello-world
    jobName: hello
  providerConfigRef:
    name: default
```

Builds are run-to-completion: they trigger once on creation, track status to terminal, and become immutable. Set `spec.forProvider.abort: true` to abort a running build.

### 6. Manage Workers

```yaml
apiVersion: ci.concourse.crossplane.io/v1alpha1
kind: Worker
metadata:
  name: worker-1
spec:
  forProvider:
    workerName: worker-1
    desiredState: landing  # landing | retiring
  providerConfigRef:
    name: default
```

Workers are observed (they self-register with Concourse). Set `desiredState` to `landing` or `retiring` to manage lifecycle.

## Development

```sh
# Run tests
go test ./...

# Lint
golangci-lint run ./...

# Regenerate CRDs + deepcopy + managed methods
make generate

# Run locally against a cluster
go run ./cmd/provider/ --debug
```

### Prerequisites
- Go 1.24+
- A Kubernetes cluster with Crossplane installed
- A running Concourse CI instance

## License

Apache 2.0 — see [LICENSE](LICENSE).
