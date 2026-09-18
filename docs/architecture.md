# Repository architecture

This document describes the current code boundaries in the Navire repository. It is intended for contributors working on the implementation in the `staging` branch.

The public product model is described in [the overview](overview.md).

## Repository organization

```text
cmd/
  core/                 Core executable entrypoint
  dispatcher/            Dispatcher executable entrypoint
  navirectl/             CLI executable entrypoint

core/
  app/                   Core application assembly
  internal/api/          HTTP server, routes, middleware, and handlers
  internal/ingest/       File-based configuration loading and watching
  internal/provider/     Core-side provider preparation
  internal/runtime/      Core lifecycle and workers
  internal/metrics/      Core metrics contracts and Prometheus exporter

dispatcher/
  internal/              Dispatcher runtime, consumers, sinks, and sender

shared/
  config/                Environment configuration helpers
  crypto/                Key and secret parsing
  heartbeat/              Heartbeat contracts
  lifecycle/              Lifecycle event contracts
  models/                 Shared event and execution models
  providers/              Provider identifiers and catalog metadata
  redis/                  Redis and Valkey configuration and client setup
  reporting/              Dispatcher reporting contracts
  runtime/                Shared runtime primitives
  template/               Template schema, parsing, validation, and catalog
  transport/              Transport contracts and Redis Streams adapter
  validation/             Shared validation rules
  version/                Build and version metadata

navirectl/
  internal/                CLI commands, middleware, runtime, and handlers
```

## Responsibility boundaries

Core decides what should happen. It owns event validation, template resolution, routing, target preparation, execution-plan publication, and aggregation of operational reports.

The Dispatcher performs delivery. It consumes an execution plan, renders the selected variant with the supplied data, formats a provider-specific request through a sink, sends the request through the sender, and reports the result.

Shared packages contain contracts and logic that must have the same meaning in more than one executable. They must not depend on Core or Dispatcher internals.

The CLI is a client-side tool. It may parse and validate declarative files locally using shared contracts, but server-side authorization and business policy remain authoritative in Core.

## Dependency rules

- shared packages must not import Core or Dispatcher internals
- Core must not depend on Dispatcher implementation packages
- Dispatcher must consume execution plans instead of rebuilding templates or routing decisions
- provider sinks must format requests and must not own delivery policy
- the sender must own the sending lifecycle and retry execution
- HTTP handlers must delegate business behavior to services
- configuration parsing belongs in configuration packages, not in business services

## Validation and tests

Pure template and event contract validation is shared by Core and `navirectl` to prevent the two entry points from accepting different data. Core still performs final validation with runtime context.

Packages should keep unit tests close to the behavior they verify. Changes to a component should run the checks for the affected scope, while shared package changes require broader checks because they can affect every executable.

## Evolution

Directory names and package layouts may change as the implementation evolves. Responsibility boundaries are more important than preserving a particular tree. New transports, administration APIs, persistence, and access-control features must be added without moving provider-specific or transport-specific behavior into event contracts.
