---
title: Integrating with Prometheus
menu_order: 90
---

Flux has a two-part integration with
[Prometheus](https://github.com/prometheus/prometheus): firstly, the
Flux daemon `fluxd` exposes metrics that Prometheus can scrape; and
secondly, the web dashboard queries those metrics to populate its
charts and gauges.

## Exposing stats to Prometheus from fluxd

Prometheus needs a way to discover all of the hosts running fluxd, so
that it can probe them for metrics.  The Docker image
`weaveworks/flux-prometheus-etcd` provides a Prometheus server that is
customized to automatically discover `fluxd` instances via etcd.  You
don't need to supply any options to `fluxd` to enable the integration
with Prometheus, although you can customize the port number on which
`fluxd` listens for connections from Prometheus with the
`--listen-prometheus` option (it defaults to port 9000).

Apart from the enhancements to support discovery via etcd,
`weaveworks/flux-prometheus-etcd` is just a plain Prometheus server.
If you already have a Prometheus server deployed, you can use that.
Though you'll need to arrange to use one of Prometheus' service
discovery mechanisms for it to scrape metrics from all hosts running
`fluxd`.


