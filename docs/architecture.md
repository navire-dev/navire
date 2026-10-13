# Repository architecture

This document describes the current code boundaries in the Navire repository. It is intended for contributors working on the implementation in the `staging` branch.

The public product model is described in [the overview](overview.md).

## Repository organization

```text
cmd/
  core/                 Core executable entrypoint
  navirectl/            CLI executable entrypoint

core/
  app/                  Core application assembly
  internal/api/         HTTP server, routes, middleware, and handlers
  internal/config/      Core configuration loading and validation
  internal/crypto/      Core encryption and key handling
  internal/dispatcher/  Core-owned delivery unit, renderer, sinks, and sender
  internal/ingest/      File-based target loading and watching
  internal/lifecycle/   Core lifecycle event definitions
  internal/metrics/     Core metrics contracts and Prometheus exporter
  internal/provider/    Core provider registry and endpoint preparation
  internal/redis/       Core Redis or Valkey client setup
  internal/runtime/     Core lifecycle, plan consumption, and worker pool
  internal/transport/   Core transport contracts and Redis Streams adapter
  internal/validation/  Core-facing validation integration

shared/
  config/               Environment configuration helpers
  logger/               Shared logging setup
  models/               Shared event, endpoint, notification, and execution-plan models
  providers/            Provider identifiers, catalog, and activation policy
  template/             Template schema, parsing, validation, routing, and catalog
  validation/           Shared validation rules
  version/              Build and version metadata

navirectl/
  app.go                CLI application assembly
  internal/              CLI commands, middleware, runtime, and handlers
```

`cmd/core/` and `cmd/navirectl/` are thin executable entrypoints. The implementation of the CLI lives in the top-level `navirectl/` package.

## Responsibility boundaries

Core owns the complete runtime path. It accepts events, validates them, resolves templates and routing, prepares provider targets, publishes immutable execution plans, consumes those plans, delivers notifications, records delivery outcomes, and acknowledges the transport messages.

The dispatcher is an internal delivery unit inside Core. It is not a separate executable or process. The Core runtime starts a bounded pool of sequential plan workers; each worker passes one execution plan through the renderer, provider sink, and common sender before Core records the result and acknowledges the transport message.

Shared packages contain contracts and logic that must have the same meaning across the repository. In particular, models, provider identifiers, and template contracts form the boundary between Core and `navirectl`. Shared packages must not depend on Core or `navirectl` internals.

The CLI is a client-side tool. It may parse and validate declarative files locally using shared contracts, but server-side authorization and business policy remain authoritative in Core.

## Delivery path

```text
producer
   |
   | POST /api/v1/events
   v
Core API
   |
   | validate, resolve, publish
   v
Redis or Valkey Streams
   |
   | provider-partitioned execution plans
   v
Core plan worker pool
   |
   | render -> sink -> sender
   v
provider endpoint
```

The current transport uses one stream per active provider, using names such as `navire:plans:discord` and `navire:plans:slack`. A shared consumer group allows the Core worker pool to consume plans from all provider streams while keeping provider queues independently selectable.

## Dependency rules

- shared packages must not import Core or `navirectl` internals
- Core API code must delegate business behavior to services
- the Core dispatcher must consume execution plans instead of rebuilding templates or routing decisions
- provider sinks must format requests and must not own the common delivery lifecycle
- the sender must own HTTP delivery, retry execution, and normalized delivery errors
- transport-specific behavior belongs in transport adapters
- configuration parsing belongs in configuration packages, not in business services

The dispatcher is intentionally kept behind an internal service boundary. If it becomes useful to run delivery in a separate process again, the delivery unit can be extracted without moving template resolution or routing back into the worker.

## Validation and tests

Pure template and event contract validation is shared by Core and `navirectl` to prevent the two entry points from accepting different data. Core still performs final validation with runtime context.

Packages should keep unit tests close to the behavior they verify. Changes to a component should run the checks for the affected scope, while shared package changes require broader checks because they can affect every executable.

## Evolution

Redis or Valkey Streams is the current transport implementation, not the final architectural boundary. New transport adapters must preserve the event, execution-plan, lifecycle, delivery-result, and message-identity contracts.

The current runtime is designed around one Core instance. PostgreSQL-backed administration, authentication, recovery workflows, additional transports, and high availability remain future work described in the [roadmap](roadmap.md).
