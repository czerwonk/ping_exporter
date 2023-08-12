# ping_exporter
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
  - google.com:
      asn: 15169

dns:
  refresh: 2m15s
  nameserver: 1.1.1.1

ping:
  interval: 2s
  timeout: 3s
  history-size: 42
  size: 120

options:
  disableIPv6: false
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

## Docker

https://hub.docker.com/r/czerwonk/ping_exporter

To run the ping_exporter as a Docker container, run:

```console
$ docker run -p 9427:9427 -v /path/to/config/directory:/config:ro --name ping_exporter czerwonk/ping_exporter
```

## Kubernetes
To run the ping_exporter in Kubernetes, you can use the supplied helm chart

### Prerequisites

* Helm v3.0.0+

### Installing the chart

To install the chart with the release name `ping-exporter`:
```console
$ helm repo add ping-exporter "https://raw.githubusercontent.com/czerwonk/ping_exporter/main/dist/charts/"
"ping-exporter" has been added to your repositories

$ helm repo update
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "ping-exporter" chart repository
Update Complete. ⎈Happy Helming!⎈

$ helm install ping-exporter ping-exporter/ping-exporter
NAME: ping-exporter
...

```

### General parameters
| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | [Affinity] |
| args | list | `[]` | Add additional [command-line arguments] when running ping_exporter |
| config | object | see [values.yaml] | Contains the contents of ping_exporter's [config file] |
| fullnameOverride | string | `""` | String to fully override `"ping-exporter.fullname"` |
| image.repository | string | `"czerwonk/ping_exporter"` | String to override the docker image repository |
| image.pullPolicy | string | `"IfNotPresent"` | String to override the pullPolicy |
| image.tag | string | `""` | Overrides the ping_exporter image tag whose default is the chart `appVersion` |
| imagePullSecrets | list | `[]` | If defined, uses a secret to pull an image from a private Docker registry or repository |
| ingress.enabled | bool | `false` | Enable an ingress resource for the ping_exporter |
| ingress.className | string | `""` | Defines which ingress controller will implement the resource |
| ingress.annotations | object | `{}` | Annotations to be added to the ingress resource |
| ingress.hosts | list | `[{"host": "chart-example.local", "paths":[{"path": "/", "pathType": "ImplementationSpecific"}]}]` | Defines the [ingress] hosts and path to proxy |
| ingress.tls | list | `[]` | Defines the secret(s) containing TLS certs for the [ingress] host |
| nameOverride | string | `""` | Provide a name in place of `ping-exporter` |
| podAnnotations | object | `{}` | Annotations to be added to ping_exporter pods |
| podSecurityContext | object | `{}` | Sets the container-level security context |
| replicaCount | number | `1` | Override the number of replicas running |
| resources | object | `{}` | Defines the ping_exporter pod's resource cpu/memory limits and requests |
| nodeSelector | object | `{}` | [Node selector] |
| securityContext.capabilities | object | `{"add": ["CAP_NET_RAW"]}` | This object overrided the pod's security context capabilities |
| service.type | string | `"ClusterIP"` | Sets the type of kubernetes service which is created for ping_exporter |
| service.port | number | `9427` | Sets the port in which the kubernetes service will listen on and communicate with the ping_exporter pod |
| service.annotations | object | `{}` | Annotations applied to the kubernetes service |
| serviceAccount.create | bool | `true` | Create a service account for the application  |
| serviceAccount.annotations | object | `{}` | Annotations applied to created service account |
| serviceAccount.name | string | `""` | Overrides the application's service account name which defaults to `"ping-exporter.fullname"` |
| tolerations | list | `[]` | [Tolerations] | 


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

[Node selector]: https://kubernetes.io/docs/user-guide/node-selection/
[Ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
[Tolerations]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
[Affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
[command-line arguments]: https://github.com/czerwonk/ping_exporter#different-time-unit
[config file]: https://github.com/czerwonk/ping_exporter#config-file
[values.yaml]: https://github.com/czerwonk/ping_exporter/blob/main/dist/charts/ping-exporter/values.yaml
