# ping_exporter
[![Test results](https://github.com/github.com/czerwonk/ping_exporter/workflows/Test/badge.svg)](https://github.com/github.com/czerwonk/ping_exporter/actions?query=workflow%3ATest)
[![Docker Build Status](https://img.shields.io/docker/cloud/build/czerwonk/ping_exporter.svg)](https://hub.docker.com/r/czerwonk/ping_exporter/builds)
[![Go Report Card](https://goreportcard.com/badge/github.com/czerwonk/ping_exporter)](https://goreportcard.com/report/github.com/czerwonk/ping_exporter)

Prometheus exporter for ICMP echo requests using https://github.com/digineo/go-ping

This is a simple server that scrapes go-ping stats and exports them via HTTP for
Prometheus consumption. The go-ping library is build and maintained by Digineo GmbH.
For more information check the [source code][go-ping].

[go-ping]: https://github.com/digineo/go-ping

## Getting Started

### Config file

Targets can be specified in a YAML based config file:

```yaml
targets:
  - 8.8.8.8
  - 8.8.4.4
  - 2001:4860:4860::8888
  - 2001:4860:4860::8844
  - google.com

dns:
  refresh: 2m15s
  nameserver: 1.1.1.1

ping:
  interval: 2s
  timeout: 3s
  history-size: 42
  payload-size: 120
```

Note: domains are resolved (regularly) to their corresponding A and AAAA
records (IPv4 and IPv6). By default, `ping_exporter` uses the system
resolver to translate domain names to IP addresses. You can override the
resolver address by specifying the `--dns.nameserver` flag when starting
the binary, e.g.

```console
$ # use Cloudflare's public DNS server
$ ./ping_exporter --dns.nameserver=1.1.1.1:53 [other options]
```

### Exported metrics

- `ping_rtt_best_seconds`:          Best round trip time in seconds
- `ping_rtt_worst_seconds`:         Worst round trip time in seconds
- `ping_rtt_mean_seconds`:          Mean round trip time in seconds
- `ping_rtt_std_deviation_seconds`: Standard deviation in seconds
- `ping_loss_ratio`:                Packet loss as a value from 0.0 to 1.0

Each metric has labels `ip` (the target's IP address), `ip_version`
(4 or 6, corresponding to the IP version), and `target` (the target's
name).

Additionally, a `ping_up` metric reports whether the exporter
is running (and in which version).

### Shell

To run the exporter:

```console
$ ./ping_exporter [options] target1 target2 ...
```

or

```console
$ ./ping_exporter --config.path my-config-file [options]
```

Help on flags:

```console
$ ./ping_exporter --help
```

Getting the results for testing via cURL:

```console
$ curl http://localhost:9427/metrics
```

### Running as non-root user

On Linux systems `CAP_NET_RAW` is required to run `ping_exporter` as unpriviliged user.
```console
# setcap cap_net_raw+ep /path/to/ping_exporter
```

When run through a rootless Docker implementation on Linux, the flag `--cap-add=CAP_NET_RAW` should be added to the `docker run` invocation.

If being invoked via systemd, you can alternately just add the following
settings to the service's unit file in the `[Service]` section:

```console
CapabilityBoundingSet=CAP_NET_RAW
AmbientCapabilities=CAP_NET_RAW
```

### Docker

https://hub.docker.com/r/czerwonk/ping_exporter

To run the ping_exporter as a Docker container, run:

```console
$ docker run -p 9427:9427 -v /path/to/config/directory:/config:ro --name ping_exporter czerwonk/ping_exporter
```

## Changes from previous versions

### `ping_loss_ratio` vs `ping_loss_percent`

Previous versions of the exporter reported packet loss via a metric named
`ping_loss_percent`.  This was somewhat misleading / wrong, because it never
actually reported a percent value (it was always a value between 0 and 1).  To
make this more clear, and to match with [Prometheus best
practices](https://prometheus.io/docs/practices/naming/#base-units), this
metric has been renamed to `ping_loss_ratio` instead.

If you had already been using an earlier version and want to continue to record
this metric in Prometheus using the old name, this can be done using the
`metric_relabel_configs` options in the Prometheus config, like so:

```console
- job_name: "ping"
  static_configs:
    <...>
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "ping_loss_ratio"
      target_label: __name__
      replacement: "ping_loss_percent"
```

### Time units

As per the recommendations for [Prometheus best
practices](https://prometheus.io/docs/practices/naming/#base-units), the
exporter reports time values in seconds by default.  Previous versions
defaulted to reporting milliseconds by default (with metric names ending in
`_ms` instead of `_seconds`), so if you are upgrading from an older version,
this may require some adjustment.

It is possible to change the ping exporter to report times in milliseconds
instead (this is not recommended, but may be useful for compatibility with
older versions, etc).  To do this, the `metrics.rttunit` command-line switch
can be used:

```console
$ # keep using seconds (default)
$ ./ping_exporter --metrics.rttunit=s [other options]
$ # use milliseconds instead
$ ./ping_exporter --metrics.rttunit=ms [other options]
$ # report both millis and seconds
$ ./ping_exporter --metrics.rttunit=both [other options]
```

If you used the `ping_exporter` in the past, and want to migrate, start
using `--metrics.rttunit=both` now. This gives you the opportunity to
update all your alerts, dashboards, and other software depending on ms
values to use proper scale (you "just" need to apply a factor of 1000
on everything). When you're ready, you just need to switch to
`--metrics.rttunit=s` (or just remove the command-line option entirely).

### Deprecated metrics

Previous versions of this exporter provided an older form of the RTT metrics
as:

- `ping_rtt_ms`: Round trip times in millis

This metric had a label `type` with one of the following values:

- `best` denotes best round trip time
- `worst` denotes worst round trip time
- `mean` denotes mean round trip time
- `std_dev` denotes standard deviation

These metrics are no longer exported by default, but can be enabled for
backwards compatibility using the `--metrics.deprecated` command-line flag:

```console
$ # also export deprecated metrics
$ ./ping_exporter --metrics.deprecated=enable [other options]
$ # or omit deprecated metrics (default)
$ ./ping_exporter --metrics.deprecated=disable [other options]
```

## Contribute

Simply fork and create a pull-request. We'll try to respond in a timely fashion.

## License

MIT License, Copyright (c) 2018
Philip Berndroth [pberndro](https://twitter.com/pberndro)
Daniel Czerwonk [dan_nrw](https://twitter.com/dan_nrw)
