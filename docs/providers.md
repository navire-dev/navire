# Providers and endpoints

Providers are external notification systems that receive notifications. An endpoint is a configured receiver within one provider.

The current runtime supports:

- Gotify
- NTFY
- Slack
- Discord

## Delivery responsibility

Provider-specific formatting is not performed by Core and is not performed by the external provider abstraction itself. The Dispatcher owns that delivery path:

```text
execution plan
    |
    v
renderer
    |
    v
provider sink       formats the provider request
    |
    v
sender              performs the request and retry lifecycle
    |
    v
external endpoint
```

Core prepares the target information required by the Dispatcher. It resolves the provider and endpoint configuration, prepares the complete delivery URL, and includes the selected delivery policy in the execution plan.

The Dispatcher does not choose a provider or endpoint. Those decisions have already been made by Core before the execution plan is published.

## Endpoint configuration

Provider configuration is declarative. A provider file identifies the provider and defines one or more named endpoints.

```yaml
provider: gotify
enabled: true

endpoints:
  - key: operations
    name: Operations notifications
    enabled: true
    url: https://gotify.example.invalid
    auth:
      type: query
      param: token
      value: ${GOTIFY_OPERATIONS_TOKEN}
```

Endpoint keys identify receivers within a provider configuration. Human-readable names and descriptions are metadata and are not used as the technical identity.

Provider credentials must be supplied through environment expansion or another deployment secret mechanism. They must not be committed to a configuration repository.

## Provider sinks

Each supported provider has a sink implementation in the Dispatcher. A sink receives the canonical execution-plan content and produces the request representation expected by its provider.

Sinks may handle provider-specific concerns such as:

- JSON or form payload construction
- provider-specific headers
- provider-specific authentication placement
- content limits and payload shape
- provider-specific response interpretation

Sinks must not decide routing, select variants, or implement a second event model.

## Sender

The sender owns the common delivery lifecycle after a sink has prepared a request. It performs the request, applies the execution policy, records duration and outcome, and returns a normalized result to the Dispatcher runtime.

Keeping the sender separate from sinks prevents each provider implementation from inventing its own retry and error-handling behavior.

## Adding a provider

Adding a provider requires:

1. a provider identifier in the shared provider catalog
2. Core-side endpoint preparation when the provider needs it
3. a Dispatcher sink that formats the provider request
4. configuration and focused tests

The event contract, template model, routing model, and execution-plan contract should not change when a provider is added.
