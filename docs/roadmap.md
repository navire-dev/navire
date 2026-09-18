# Roadmap

This roadmap describes the intended product evolution. It is informational, not a delivery commitment, and may change as Navire is used and new operational constraints become visible.

## 0.0.1: Secure single-user MVP

The first release establishes a usable and secure vertical slice around the existing Core, transport, Dispatcher, provider, and observability model.

The target includes:

- authenticated producer access and first-owner onboarding
- PostgreSQL as the source of truth for managed identity and resources
- one default resource-ownership team without team management
- a separate authenticated Administration API
- `navirectl` as a thin Admin API client
- local validation of templates, providers, and declarative configuration
- CLI push and apply workflows through the Administration API
- provider and endpoint management
- operational metrics and lifecycle reporting

The MVP intentionally does not include team management, multi-user administration, a mandatory web UI, delivery replay, a DLQ workflow, or automatic deduplication.

## 0.0.2: Delivery reliability and mentions

This milestone focuses on making accepted delivery work recoverable and easier to operate.

Planned areas include durable publication recovery, explicit terminal failures, DLQ semantics, replay, audit history, and logical mentions that can be resolved by provider integrations.

## 0.0.3: Multi-user security and declarative workflows

This milestone expands the single-user model with users, teams, memberships, ACL groups, direct rules, and team-scoped resource administration.

It also extends `navirectl` with remote diff, CI-oriented JSON output, and a dedicated container image for declarative configuration workflows.

## 0.0.4: Transport independence

Redis or Valkey is the first transport implementation, not the final architectural boundary. This milestone adds further pub/sub transport adapters while preserving Navire's event, execution-plan, heartbeat, lifecycle, and reporting contracts.

## 0.1: High availability and operational scale

This milestone addresses multi-Core coordination, shared operational state, failover behavior, backup and migration procedures, and production-scale deployment concerns.

## 1.0: Stable public contracts

Version 1.0 should provide stable API, configuration, execution-plan, provider, upgrade, and operational contracts with documented compatibility and migration guarantees.
