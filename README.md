# ping_exporter
[![Build Status](https://travis-ci.org/czerwonk/ping_exporter.svg)](https://travis-ci.org/czerwonk/ping_exporter)
[![Docker Build Statu](https://img.shields.io/docker/build/czerwonk/ping_exporter.svg)](https://hub.docker.com/r/czerwonk/ping_exporter/builds)
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

- `ping_rtt_best_ms`:          Best round trip time in millis
- `ping_rtt_worst_ms`:         Worst round trip time in millis
- `ping_rtt_mean_ms`:          Mean round trip time in millis
- `ping_rtt_std_deviation_ms`: Standard deviation in millis
- `ping_loss_percent`:         Packet loss in percent

Each metric has labels `ip` (the target's IP address), `ip_version`
(4 or 6, corresponding to the IP version), and `target` (the target's
name).

Additionally, a `ping_up` metric reports whether the exporter
is running (and in which version).

#### Deprecated metrics

- `ping_rtt_ms`: Round trip trim in millis

This metric has a label `type` with one of the following values:

- `best` denotes best round trip time
- `worst` denotes worst round trip time
- `mean` denotes mean round trip time
- `std_dev` denotes standard deviation

These metrics are exported by default, but this may change with a future
release of this exporter.

To ensure forward- or backward compatability, use the `--metrics.deprecated`
flag:

```console
$ # also export deprecated metrics
$ ./ping_exporter --metrics.deprecated=enable [other options]
$ # or omit deprecated metrics
$ ./ping_exporter --metrics.deprecated=disable [other options]
```

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

### Docker

https://hub.docker.com/r/czerwonk/ping_exporter

To run the ping_exporter as a Docker container, run:

```console
$ docker run -p 9427:9427 -v ./config:/config:ro --name ping_exporter czerwonk/ping_exporter
```


## Contribute

Simply fork and create a pull-request. We'll try to respond in a timely fashion.

## License

MIT License, Copyright (c) 2018
Philip Berndroth [pberndro](https://twitter.com/pberndro)
Daniel Czerwonk [dan_nrw](https://twitter.com/dan_nrw)
