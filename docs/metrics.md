# Navire metrics

Core exposes a Prometheus-compatible metrics endpoint at `GET /metrics`.

Dispatcher instances do not expose a metrics HTTP server in the current implementation. They publish heartbeats and delivery reports through the configured transport, and Core aggregates those observations.

The endpoint is intentionally protected by deployment-level network controls rather than application authentication in the current runtime.

## Naming

Application metrics use the `navire_` prefix. Counters use the `_total` suffix. The application exports factual counters and gauges. Prometheus or Grafana should calculate rates, increases, and ratios over a selected time window.

## Event metrics

```text
navire_events_received_total{component="core"}
navire_events_accepted_total{component="core"}
navire_events_rejected_total{component="core",reason}
navire_events_published_total{component="core"}
navire_events_publish_errors_total{component="core"}
navire_events_total{component="core",template,variant,state}
```

`navire_events_total` counts accepted events grouped by the state declared by the template variant. The current states are `success`, `failure`, and `informational`.

Event state is business meaning. It is not delivery outcome. A `success` event can fail to reach a provider, and an `informational` event is not automatically a successful delivery.

Rejection reasons are bounded categories such as `template_load`, `variant_not_found`, `template_data`, `no_active_targets`, and `publish`.

## Delivery metrics

```text
navire_delivery_attempts_total{component="core",provider,state}
navire_delivery_duration_seconds{component="core",provider}
```

Delivery state describes the result of a target-specific attempt. It is independent from the event state. The current delivery states are `success`, `failure`, and `skipped`.

## Runtime metrics

```text
navire_up{component="core"}
navire_uptime_seconds{component="core"}
navire_redis_connected{component="core"}
navire_redis_operation_errors_total{component="core",operation}
navire_stream_messages_published_total{component="core"}
navire_stream_messages_consumed_total{component="core"}
navire_stream_messages_acked_total{component="core"}
navire_dispatcher_up{component="core",dispatcher}
navire_dispatcher_redis_connected{component="core",dispatcher}
navire_stream_consumer_lag{component="core",dispatcher}
```

`navire_dispatcher_up`, `navire_dispatcher_redis_connected`, and `navire_stream_consumer_lag` are derived from fresh Dispatcher heartbeats. When a Dispatcher stops sending heartbeats, its stale labelled series are removed by Core rather than being kept indefinitely.

## Querying counters

Counters should be queried with a range function when the dashboard needs activity over time:

```promql
sum(increase(navire_events_received_total{job="navire-core-prod"}[$__range]))
```

For a current cumulative value, query the counter directly:

```promql
navire_events_received_total{job="navire-core-prod"}
```

Grafana dashboards should not hard-code a single Core or Dispatcher identifier. Use the labels exposed by Prometheus to build variables and aggregate over the selected installation.

## Label policy

Labels must remain bounded and operationally useful. Provider identifiers, event states, rejection reasons, and configured Dispatcher identifiers are controlled dimensions. Arbitrary payload values must never become metric labels.
