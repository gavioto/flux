---
title: Flux overview
menu_order: 20
---

This page describes the main concepts needed to use Flux.

## Key concepts

Weave Flux lets you define _services_.  A service has an IP address
and port.  These service addresses are _floating addresses_ and they
don't correspond to any host.  When clients attempt to connect to one,
the Flux daemon transparently forwards the connection to a service
_instance_.  Connections are load balanced over the available
instances.

_Instances_ correspond to Docker containers.  The containers are
automatically enrolled as service instances according to _selection
rules_ you supply.  For example, a selection rule might specify that
all containers with a particular image name become instances of a
corresponding service.

## Components

A running Flux deployment consists of

 1. a [fluxd daemon](/site/daemon.md) on each host, which detects
    instances starting and stopping on that host, and proxies
    connections to services
 2. Optionally, one or more [edge balancers](/site/edgebal.md), which
    accept connections to services from the outside world.

To control and examine the state of your services, Flux provides a
command-line tool called [fluxctl](/site/fluxctl.md). To monitor the
performance of the services, Flux has a [web dashboard](/site/web.md).

All of the above are available as Docker images.

At present, Flux relies on [etcd][etcd-site] to store its
configuration and runtime data; and may be used [in
conjunction](/site/prometheus.md) with [Prometheus][prometheus-site]
to provide runtime metrics for services.

[etcd-site]: https://github.com/coreos/etcd
[prometheus-site]: https://github.com/prometheus/prometheus


**See Also**

 * [ The Flux daemon, fluxd](/site/daemon.md)
 * [Edge-balancer Docker Image](/site/edgebal.md)
 * [fluxctl command-line interface](/site/fluxctl.md)
 * [Integrating with Prometheus](/site/prometheus.md)