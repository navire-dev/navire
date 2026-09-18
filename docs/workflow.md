# Event lifecycle

This document describes the current event and delivery workflow implemented by Navire.

## Ingestion endpoint

Producers submit operational events to:

```text
POST /api/v1/events
Content-Type: application/json
```

The endpoint accepts a JSON object with the following fields:

```json
{
  "key": "fake_cicd",
  "variant": "deployment_failed",
  "priority": "high",
  "idempotency_key": "deployment-123",
  "timestamp": "2026-09-17T12:00:00Z",
  "data": {
    "service": "example-api",
    "version": "1.4.2"
  }
}
```

`key` identifies the template. `variant` selects the variant when the template contains more than one applicable variant. `priority` and `idempotency_key` are optional overrides. `timestamp` is accepted as part of the contract but Core assigns the processing timestamp for the current execution path. `data` contains the values required by the selected template.

Unknown JSON fields are rejected. The template key must contain between one and fifty lowercase letters, digits, hyphens, or underscores, and must start with a letter or digit.

## Core processing

Core performs the following operations:

1. validates the request shape and field limits
2. resolves the template by its key
3. resolves the requested or matching variant
4. validates the variant state, priority, and required data fields
5. evaluates the template routing rules
6. resolves active provider endpoints
7. creates one immutable execution plan per selected target
8. publishes the plans to the configured transport
9. returns `202 Accepted` after publication succeeds

The event response contains the generated idempotency or message identifier and the published stream identifiers.

An event that cannot be validated, resolved, routed, or published is rejected with an error response and a bounded rejection reason in the metrics.

## Execution plan

An execution plan is a target-specific snapshot. It contains the selected variant, its title and body, the event data, the resolved provider target, and the delivery policy.

The Dispatcher does not load the original template to rebuild the plan. This keeps Core responsible for event interpretation and makes the plan the boundary between decision-making and delivery.

For example, an execution plan for the `operations` Gotify endpoint may look like this:

```json
{
  "schema_version": 2,
  "message_id": "event-2026-09-17-001",
  "idempotency_key": "event-2026-09-17-001:gotify_operations",
  "created_at": "2026-09-17T16:50:57.573871515+02:00",
  "variant": {
    "name": "deployment_failed",
    "title": "Deployment failed for {{ .service }}",
    "body": "**Version:** `{{ .version }}`\n**Environment:** `{{ .environment }}`\n**Target:** `{{ .target }}`\n**Error:** `{{ .error_code }}`\n**Rollback:** `{{ .rollback }}`\n**Pipeline:** {{ .pipeline_url }}",
    "priority": "high",
    "state": "failure"
  },
  "data": {
    "service": "example-api",
    "version": "1.4.2",
    "environment": "production",
    "target": "production-eu",
    "error_code": "DEPLOYMENT_TIMEOUT",
    "rollback": "automatic",
    "pipeline_url": "https://ci.example.invalid/pipelines/deployment-123"
  },
  "target": {
    "name": "operations",
    "provider": "gotify",
    "url": "ENC[encrypted resolved endpoint URL]",
    "policy": {
      "timeout": "5s",
      "retry": {
        "max_attempts": 3,
        "backoff": "exponential",
        "initial_wait": "5s"
      }
    }
  }
}
```

The encrypted target URL is an internal runtime value. It is shown here only to illustrate the execution-plan boundary and must never be replaced with a real credential in documentation, logs, or tests.

## Dispatcher processing

A Dispatcher consumes one execution plan and:

1. renders the selected title and body with the plan data
2. passes the rendered content to the provider sink
3. sends the formatted request through the common sender
4. applies the target delivery policy
5. publishes a normalized delivery report

The report allows Core to update delivery metrics and operational Dispatcher state.

## Configuration

The current implementation loads templates and provider endpoint definitions from configured directories. The template loader is recursive and rejects invalid files rather than silently accepting unknown fields.

The CLI can validate the same declarative template contracts locally. Local validation is useful feedback, but Core remains authoritative when an event is ingested.

## Related endpoints

The current Core server also exposes:

- `GET /healthz` for liveness
- `GET /readyz` for readiness
- `GET /metrics` for Prometheus-compatible metrics
- `POST /api/register-dispatcher` for Dispatcher registration

Human authentication, the Administration API, and persistent resource management are future milestones of the MVP and are not part of this current ingestion workflow.
