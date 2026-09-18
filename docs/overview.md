# Navire overview

Navire is a self-hosted platform for delivering operational events to notification systems. It separates event producers from notification delivery so that an application or automation system does not need to know which provider will receive an event.

## The problem

Operational systems produce events about deployments, services, storage, automation, and security. Without a delivery layer, each producer tends to contain provider-specific payloads, credentials, routing decisions, and retry behavior.

That coupling makes notification infrastructure difficult to change. Replacing one provider can require changes in many unrelated producers.

## The Navire model

Producers submit an event using the Navire event contract. Core validates the event, resolves a template variant and its configured targets, and publishes one execution plan per target. Dispatchers consume those plans and deliver notifications independently of Core.

```text
producer
   |
   |  POST /api/v1/events
   v
Core
   |  validate, resolve, publish
   v
pub/sub transport
   v
Dispatcher
   |  render, format, send, report
   v
provider endpoint
```

## Current components

### Core

Core is the decision-making component. It accepts events, validates their shape and content, resolves templates and variants, applies routing rules, prepares provider targets, and publishes immutable execution plans.

Core also aggregates Dispatcher reports and exposes the Prometheus-compatible metrics endpoint.

### Dispatchers

Dispatchers are independent delivery workers. A Dispatcher consumes target-specific execution plans, renders the already selected variant with event data, formats the provider request through a provider sink, sends it, and reports the result back to Core through the transport.

Dispatchers do not select templates, reinterpret routing rules, or make business decisions about the event.

### Templates

Templates describe the operational meaning and presentation of an event. Each template has a stable key, human-readable metadata, and one or more variants. A variant declares its state, priority, title, body, and the fields it expects in event data.

The event references a template by its key. It does not contain provider-specific formatting.

### Providers and endpoints

A provider identifies an external notification system such as Gotify, NTFY, Slack, or Discord. An endpoint is a configured receiver within that provider. Core resolves endpoint configuration and passes the resulting target to a Dispatcher.

The Dispatcher owns provider-specific request formatting. The provider itself is the external system receiving that request.

## Transport

Redis or Valkey Streams is the transport used by the current implementation. The event, execution-plan, heartbeat, lifecycle, and reporting contracts are independent of that choice.

Future transport adapters may support other pub/sub backends without changing the contracts between producers, Core, and Dispatchers.

## Configuration model

The current runtime loads templates and provider endpoint configuration from files. The CLI can validate declarative configuration locally. The server remains responsible for final runtime validation and execution.

See the [declarative configuration reference](configuration.md) for the template and provider file formats.

An authenticated administration API and a persistent control plane are planned for the MVP target. They are not part of the current file-based runtime described by this document.

The current file-based runtime configuration is an interim implementation. Over time, it will be replaced by a repository-based configuration workflow managed through navirectl, with the CLI becoming the primary interface for validating and applying configuration to Navire.

## Operational model

Core and Dispatchers are separate processes. Multiple Dispatchers can consume plans from the same transport and report their state independently. Core exposes health, readiness, metrics, and Dispatcher lifecycle information for operational monitoring.

The current system is designed around one Core instance. Active-active Core behavior and high availability are future work, not current guarantees.
