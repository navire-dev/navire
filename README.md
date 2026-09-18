# Navire

> **Operational events should not know how they are delivered.**

Navire is an open-source, self-hosted platform for delivering operational events to notification providers.

It separates the systems that produce events from the systems that deliver notifications. Producers describe what happened. Navire applies the configured templates and routing rules, then delivers the resulting notification to one or more operational destinations.

## Why Navire exists

Operational systems generate events constantly:

- a deployment succeeds or fails
- a service becomes unavailable or recovers
- storage reaches a critical threshold
- an automation job detects drift
- a security system reports an incident

Without a delivery layer, every producer has to know the notification provider, construct provider-specific payloads, store or access provider credentials, and often implement its own retry behavior. That couples operational software to the notification infrastructure around it.

Changing the provider then means changing every producer that sends to it.

Navire introduces a boundary between those responsibilities. Producers submit operational events using a stable contract, while Navire owns the decisions that follow: presentation, target selection, delivery, and delivery reporting.

## The core model

Navire is built around a small set of responsibilities:

- **Producers** report operational events
- **Core** validates events, resolves templates and targets, and publishes execution plans
- **Redis or Valkey Streams** provide the initial pub/sub transport implementation for execution plans and lifecycle data
- **Dispatchers** consume execution plans, translate them into the provider-specific request format, and deliver notifications independently of Core
- **Providers** represent the external notification systems that receive those requests

Provider-specific formatting is an implementation detail of the Dispatcher. Producers do not need to know whether a notification is eventually received by Gotify, NTFY, Slack, Discord, or another supported provider.

This separation also allows multiple Dispatchers to operate independently and keeps the delivery path separate from the event-ingestion path.

## What Navire is for

Navire is intended for operational environments such as:

- homelabs and self-hosted infrastructure
- platform and operations teams
- CI/CD systems
- monitoring and automation platforms
- security operations
- companies operating services and critical workloads

It is not a newsletter system, a marketing platform, or a general-purpose messaging application. Its scope is operational events and the reliable, provider-independent delivery of those events.

## Project status

Navire is under active development and is not yet a stable production release.

The `main` branch contains the public project entry point and community materials. Active implementation work is carried out on the `staging` branch until it is ready to be presented as a release.

The current development direction focuses on:

- a secure single-user MVP
- authenticated event ingestion
- declarative templates and provider configuration
- Core-to-Dispatcher execution plans
- provider-independent delivery
- operational metrics and lifecycle reporting
- a CLI for validating and managing declarative configuration, designed for interactive use and CI/CD automation, with a container image planned for pipeline execution

PostgreSQL-backed administration, richer access control, delivery recovery, replay, deduplication, and additional transport implementations are planned for later milestones. They are not represented as completed capabilities here.

This README describes the long-term direction and the product model Navire is being built towards. The public technical documentation focuses on the current implementation and the contracts targeted for the MVP. Future architecture and detailed design decisions remain documented separately until they become part of a public release.

## Design principles

Navire follows a few principles throughout its development:

- event producers should not contain provider-specific delivery logic
- Core decides what should happen, while Dispatchers perform delivery
- provider-specific formatting and behavior stay behind explicit contracts
- configuration and business rules should remain declarative where possible
- transport backends should not redefine Navire's event and execution contracts
- observability is part of the delivery design, not an afterthought
- reliability guarantees must be explicit rather than implied by an HTTP response

The project favors a small number of clearly separated components over a single service that owns ingestion, rendering, provider integration, and delivery retries all at once.

## Roadmap

Navire is developed incrementally. The roadmap describes the intended evolution of the project, not a fixed delivery commitment. Priorities may change as the system is used, tested, and exposed to new operational constraints.

The planned progression is:

1. establish a secure and usable single-user foundation
2. make delivery durable, observable, and recoverable
3. introduce administration, users, teams, and access control
4. support additional pub/sub backends without changing Navire's event and execution contracts
5. address high availability and broader operational scale
6. stabilize long-term compatibility and public contracts

See the [project roadmap](docs/roadmap.md) for the current planned scope.

## Contributing

Navire is released under the [GNU Affero General Public License v3.0](LICENCE).

Contributions will be welcome after the first stable release. Until then, the project is being prepared for its initial public release and external pull requests are not being accepted.

Please read the [contribution guidelines](CONTRIBUTING) before proposing work after the first release.

Security issues should be reported privately according to the [security policy](.github/SECURITY.md).

## License

Navire is free software distributed under the terms of the GNU Affero General Public License version 3. See [LICENCE](LICENCE) for the complete license text.
