# Navire overview

Navire is a self-hosted platform for delivering operational events to notification systems. It separates event producers from notification delivery so that an application or automation system does not need to know which provider will receive an event.

## The problem

Operational systems produce events about deployments, services, storage, automation, and security. Without a delivery layer, each producer tends to contain provider-specific payloads, credentials, routing decisions, and retry behavior.

That coupling makes notification infrastructure difficult to change. Replacing one provider can require changes in many unrelated producers.

## The Navire model

Producers submit an event using the Navire event contract. Core validates the event, resolves a template variant and its configured targets, and publishes one execution plan per target. Core then consumes those plans through its internal dispatcher worker pool and delivers them independently.

```text
producer
   |
   |  POST /api/v1/events
   v
Core API
   |  validate, resolve, publish
   v
Redis or Valkey Streams
   |  one stream per active provider
   v
Core dispatcher workers
   |  render, format, send, acknowledge
   v
provider endpoint
```

The dispatcher is a delivery unit inside Core, not a separate service. This keeps the current deployment simple while preserving a boundary that can be extracted later if the operational model requires it.

## Current components

### Core

Core is the decision-making and delivery component. It accepts events, validates their shape and content, resolves templates and variants, applies routing rules, prepares provider targets, publishes immutable execution plans, consumes those plans, and delivers notifications.

Core also exposes health, readiness, and Prometheus-compatible metrics. Its dispatcher workers record delivery outcomes and acknowledge plans after processing.

### Templates

Templates describe the operational meaning and presentation of an event. Each template has a stable key, human-readable metadata, and one or more variants. A variant declares its state, priority, title, body, and the fields it expects in event data.

The event references a template by its key. It does not contain provider-specific formatting.

### Providers and endpoints

A provider identifies an external notification system such as Gotify, NTFY, Slack, or Discord. An endpoint is a configured receiver within that provider. Core resolves endpoint configuration and includes the selected target in the execution plan.

Provider-specific formatting belongs to the dispatcher sink. The provider itself is the external system receiving the request.

## Dispatcher delivery unit

Each worker processes one execution plan at a time:

```text
execution plan
    |
    v
renderer          maps event data into the selected title and body
    |
    v
provider sink     builds the provider-specific request
    |
    v
sender            performs HTTP delivery and retries
    |
    v
ACK by Core       acknowledges the transport message
```

The worker pool is bounded by `NAVIRE_DISPATCHER_WORKERS`, which defaults to 20 and accepts values from 1 to 100. A failed delivery does not prevent other workers from processing other plans. The sender honors a provider-supplied `Retry-After` delay when one is returned. Full provider-aware rate limiting, adaptive concurrency, and recovery workflows are future work.

## Transport

Redis or Valkey Streams is the transport used by the current implementation. Execution plans are partitioned by provider using streams such as `navire:plans:gotify` and `navire:plans:discord`. The event, execution-plan, lifecycle, reporting, and message-identity contracts are independent of that choice.

Future transport adapters may support other pub/sub backends without changing those contracts.

## Configuration model

The current runtime loads templates and provider endpoint configuration from files. The CLI can validate declarative configuration locally. The server remains responsible for final runtime validation and execution.

See the [declarative configuration reference](configuration.md) for the template and provider file formats.

An authenticated administration API and a persistent control plane are planned for the MVP target. They are not part of the current file-based runtime.

The file-based runtime configuration is an interim implementation. Over time, it will be replaced by a repository-based configuration workflow managed through `navirectl`.

## Operational model

The current system is designed around one Core instance. Core exposes health, readiness, metrics, and delivery state for operational monitoring. Active-active Core behavior and high availability are future work, not current guarantees.
