# Navire metrics

Core exposes a Prometheus-compatible metrics endpoint at `GET /metrics`. The dispatcher is part of Core, so there is no separate dispatcher metrics server or dispatcher heartbeat metric in the current implementation.

The endpoint is protected by deployment-level network controls rather than application authentication in the current runtime.

## Naming

Application metrics use the `navire_` prefix. Counters use the `_total` suffix. The application exports factual counters, gauges, and histograms. Prometheus or Grafana should calculate rates, increases, and ratios over a selected time window.

All current metrics use the bounded constant label `component="core"`.

## Event metrics

```text
navire_events_received_total{component="core"}
navire_events_accepted_total{component="core"}
navire_events_rejected_total{component="core",reason}
navire_events_total{component="core",template,variant,state}
navire_events_published_total{component="core"}
navire_events_publish_errors_total{component="core"}
```

`navire_events_total` counts accepted events grouped by the state declared by the template variant. The current states are `success`, `failure`, and `informational`.

Event state is business meaning. It is not delivery outcome. A `success` event can fail to reach a provider, and an `informational` event is not automatically a successful delivery.

Rejection reasons are bounded categories such as `template_load`, `variant_not_found`, `template_data`, `no_active_targets`, and `publish`.

## Transport and delivery metrics

```text
navire_stream_messages_published_total{component="core"}
navire_stream_messages_consumed_total{component="core",provider}
navire_stream_messages_acked_total{component="core",provider}
navire_delivery_attempts_total{component="core",provider,state}
navire_delivery_duration_seconds{component="core",provider}
```

`navire_stream_messages_published_total` counts execution plans published by Core. Consumed and acknowledged plans are labelled by provider because the current transport uses one Redis or Valkey Stream per provider.

`navire_delivery_attempts_total` records the final delivery outcome observed by Core. Current states are `success` and `failure`. A failed delivery may still be acknowledged after its configured retry policy is exhausted; DLQ and replay semantics are future work.

`navire_delivery_duration_seconds` is a Prometheus histogram covering the complete delivery operation, including the configured retries.

## Runtime metrics

```text
navire_up{component="core"}
navire_uptime_seconds{component="core"}
navire_redis_connected{component="core"}
navire_redis_operation_errors_total{component="core",operation}
```

There are no current `navire_dispatcher_*` or per-dispatcher lag metrics. The worker pool is an internal part of Core, and its activity is represented through the provider-labelled stream and delivery metrics above.

## Querying counters

Counters should be queried with a range function when the dashboard needs activity over time:

```promql
sum(increase(navire_events_received_total{job="navire-core-prod"}[$__range]))
```

For a current cumulative value, query the counter directly:

```promql
navire_events_received_total{job="navire-core-prod"}
```

For provider delivery outcomes:

```promql
sum by (provider, state) (
  increase(navire_delivery_attempts_total{job="navire-core-prod"}[$__range])
)
```

Grafana dashboards should not hard-code a single Core identity. Use Prometheus labels such as `job`, `instance`, and `component` to build variables and aggregate over the selected installation.

## Label policy

Labels must remain bounded and operationally useful. Provider identifiers, event states, delivery states, rejection reasons, template keys, and variant names are controlled dimensions. Arbitrary payload values, message IDs, endpoint URLs, and error text must never become metric labels.
