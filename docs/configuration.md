# Declarative configuration

The current runtime uses declarative YAML files for templates and provider endpoints. The CLI can validate these files locally, while Core performs the authoritative validation when it loads configuration or accepts an event.

Configuration files may be organized in subdirectories. The loader discovers YAML files recursively.

## Template files

A template defines the event variants that a producer can reference, the content rendered for each variant, and the provider targets selected by routing rules.

The producer references a template by its `key`:

```json
{
  "key": "fake_cicd",
  "variant": "deployment_failed",
  "priority": "high",
  "data": {
    "service": "example-api",
    "version": "1.4.2",
    "environment": "production",
    "target": "production-eu",
    "error_code": "DEPLOYMENT_TIMEOUT",
    "rollback": "automatic",
    "pipeline_url": "https://ci.example.invalid/pipelines/deployment-123"
  }
}
```

A small template for that event could look like this:

```yaml
---
key: fake_cicd
name: CI/CD events
description: Deployment events emitted by a CI/CD system

variants:
  deployment_failed:
    state: failure
    priority: critical
    title: "Deployment failed for {{ .service }}"
    body: |
      **Version:** `{{ .version }}`
      **Environment:** `{{ .environment }}`
      **Target:** `{{ .target }}`
      **Error:** `{{ .error_code }}`
      **Rollback:** `{{ .rollback }}`
      **Pipeline:** {{ .pipeline_url }}

  deployment_recovered:
    state: success
    priority: normal
    title: "Deployment recovered for {{ .service }}"
    body: |
      **Version:** `{{ .version }}`
      **Environment:** `{{ .environment }}`
      **Target:** `{{ .target }}`

providers:
  defaults:
    timeout: 5s
    retry:
      backoff: exponential
      max_attempts: 3
      initial_wait: 5s
  gotify:
    endpoints:
      operations: {}
  ntfy:
    endpoints:
      operations: {}
      cicd: {}

rules:
  - condition:
      priority: critical
      data:
        environment: production
    providers:
      gotify:
        endpoints: [operations]
      ntfy:
        endpoints: [operations, cicd]

  - condition:
      priority: normal
      data:
        environment: production
    providers:
      gotify:
        endpoints: [operations]
```

The template `providers` section references endpoint keys from provider configuration. It does not contain provider credentials or provider-specific request payloads.

## Routing rules

Routing rules are optional. If a template has no rules, every enabled endpoint declared in its `providers` section is eligible for delivery.

When rules are present, Core evaluates each rule against the resolved event. A rule can match on:

- the effective priority of the event
- the variant state
- exact values in the event data

The effective priority includes any priority override supplied by the producer. Data conditions compare the configured value with the corresponding event data value.

If one or more rules match, Navire selects the destinations from the most specific matching rules. A rule becomes more specific for each priority, state, or data condition it declares. Rules with the same highest specificity are combined.

If rules are present but none matches the event, Navire falls back to all enabled destinations declared by the template. Routing rules therefore narrow the default destination set; they do not make an event disappear merely because no condition matched.

## Provider files

Provider files define the external receivers available to templates. An endpoint key identifies one receiver within a provider configuration.

```yaml
---
provider: gotify
enabled: true

endpoints:
  - key: operations
    name: Operations notifications
    enabled: true
    url: https://gotify.example.invalid/message
    auth:
      type: query
      param: token
      value: ${GOTIFY_OPERATIONS_TOKEN}
```

The same provider may be split across multiple files to keep configuration manageable. Core merges the files during loading and rejects duplicate provider and endpoint identities.

The endpoint identity is the pair:

```text
provider + endpoint key
```

For example, `gotify + operations` is different from `ntfy + operations`. The endpoint `key` is technical identity. The optional `name` is human-readable metadata.

## Authentication fields

The current configuration supports these authentication modes:

```yaml
auth:
  type: none
```

`none` means that Navire sends the request without adding authentication data. The endpoint URL may still contain information required by the provider, but Navire does not add a credential.

```yaml
auth:
  type: query
  param: token
  value: ${PROVIDER_TOKEN}
```

`query` adds the configured value to the endpoint URL as a query parameter. In this example, the final request URL contains `?token=...`. The value is expanded from the deployment environment and is encrypted before it is stored in the runtime endpoint projection.

The current implementation also supports `path_segment` for providers whose credentials are part of the URL path. Discord is an example of this pattern:

```yaml
auth:
  type: path_segment
  value: ${DISCORD_WEBHOOK_TOKEN}
```

The `key` under `endpoints` identifies the endpoint. The `param` under `auth` identifies the query parameter used by the `query` mode. They are different fields with different scopes.

Header and bearer authentication are reserved for a future end-to-end implementation. They are not currently part of the public configuration contract.

## Validation rules

The CLI and Core reject invalid configuration rather than silently ignoring it.

Important rules include:

- template and endpoint keys contain one to fifty lowercase letters, digits, hyphens, or underscores
- keys start with a letter or digit
- template variants must declare a valid event state
- template data must contain every field referenced by the selected variant
- provider identifiers must exist in the active provider catalog
- endpoint keys must be unique within a provider
- duplicate provider and endpoint identities across files are rejected
- unknown YAML fields are rejected
- provider credentials must be supplied through secret expansion and must not be committed

## Configuration workflow

The current file loader is an interim runtime mechanism. The intended workflow is repository-based configuration managed by `navirectl`:

1. initialize a configuration repository
2. organize templates and provider files
3. validate the repository locally
4. compare it with the remote Navire configuration
5. apply validated changes through the Administration API

The repository and CLI workflow is planned to replace direct production file loading over time.
