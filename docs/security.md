# Security

Navire handles operational events and provider credentials, but the current runtime is still an MVP foundation. This document describes the security controls that exist today and the limits that operators must understand.

## Current trust boundaries

The current Core server does not provide human authentication, user accounts, teams, ACLs, or an Administration API. Event ingestion and operational endpoints must therefore be protected by the deployment network, reverse proxy, firewall, or another external access-control layer.

The future Administration API will introduce a separate authentication and authorization boundary. Its design is not part of the current runtime contract.

## Event access

Event ingestion is exposed at `POST /api/v1/events`. Core can restrict requests by configured client CIDRs. When Core runs behind a reverse proxy or tunnel, trusted proxy resolution can use forwarded client information, but only when the proxy address is included in the explicit trusted-proxy allowlist.

The relevant configuration is scoped to event traffic:

```text
NAVIRE_EVENTS_TRUST_PROXY
NAVIRE_EVENTS_ALLOWED_PROXIES
NAVIRE_EVENTS_ALLOWED_CIDRS
```

These settings must match the actual network topology. Enabling proxy trust without restricting which proxies are trusted allows clients to influence the address used for access decisions.

Event rate limiting is disabled unless both of these optional variables are configured:

```text
NAVIRE_EVENTS_RATE_LIMIT_BURST
NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN
```

When enabled, Core applies a token bucket per resolved client IP to event ingestion only. The burst controls how many events can be accepted immediately, while the refill value controls the sustained rate. These values must account for legitimate producer bursts and should not be treated as a delivery scheduling mechanism.

## Dispatcher registration

Dispatchers register with Core through `POST /api/register-dispatcher`. Registration is protected by the shared Dispatcher registration secret. The secret must be treated as deployment credentials and must not be committed to configuration files or logs.

## Transport and encryption keys

The runtime uses Redis or Valkey Streams for execution plans, lifecycle events, heartbeats, and delivery reports. Transport access must be protected by the Redis or Valkey deployment configuration and network policy.

Core uses separate keys for protected runtime data:

- `NAVIRE_SECRETS_ENCRYPTION_KEY` protects configured provider secrets
- `NAVIRE_DISPATCH_PLAN_KEY` protects execution-plan data exchanged with Dispatchers
- `NAVIRE_DSPC_REGISTRATION_SECRET` authenticates Dispatcher registration

These values must be generated independently, stored outside the repository, and rotated according to the deployment's operational policy.

## Provider credentials

Provider tokens, webhook URLs, and other credentials must be supplied through environment expansion or a deployment secret mechanism. They must not be placed in public repositories, event payloads, issue reports, or dashboards.

Execution plans may contain the resolved target required by a Dispatcher. Access to the transport and its stored entries must therefore be treated as sensitive.

## Logging and observability

Logs and metrics should describe outcomes without exposing credentials or complete secret-bearing URLs. Before sharing logs in an issue, remove tokens, webhook URLs, authorization headers, and sensitive event data.

Metrics are currently exposed without application-level authentication. Restrict `/metrics`, health endpoints, and the Core listener through the deployment network or reverse proxy as appropriate.

## Security roadmap

Human authentication, the first-owner bootstrap flow, persistent identity, teams, ACLs, audit history, and a separate Administration API are planned parts of the MVP target. They must not be inferred from the current file-based runtime or treated as implemented capabilities.
