# ping_exporter
[![Build Status](https://travis-ci.org/czerwonk/ping_exporter.svg)](https://travis-ci.org/czerwonk/ping_exporter)
[![Docker Build Statu](https://img.shields.io/docker/build/czerwonk/ping_exporter.svg)](https://hub.docker.com/r/czerwonk/ping_exporter/builds)
[![Go Report Card](https://goreportcard.com/badge/github.com/czerwonk/ping_exporter)](https://goreportcard.com/report/github.com/czerwonk/ping_exporter)

Prometheus exporter for ICMP echo requests using https://github.com/digineo/go-ping

This is a simple server that scrapes go-ping stats and exports them via HTTP for
Prometheus consumption. The go-ping library is build and maintained by Digineo GmbH.
For more information check the [source code][go-ping].

[go-ping]: https://github.com/digineo/go-ping

## Remarks

# Version >= 3.0
Since [Prometheus best practices][prom-bcp] suggest to use labels for a specific type of metric (eg. avg, worst, ...) we renamed the following metrics to one single metric (`ping_rtt_ms`) and added an additional `type` label:
* ping_best_ms
* ping_worst_ms
* ping_mean_ms
* ping_std_deviation_ms

Please update your recording rules and dashboards accordingly when updating.

[prom-bcp]: https://prometheus.io/docs/practices/instrumentation/#use-labels

## Getting Started

### Shell

To run the exporter via:

```bash
./ping_exporter [options] target1 target2 ...
```

Help on flags:

```bash
./ping_exporter --help
```

Getting the results for testing via cURL:

```bash
curl http://localhost:9427/metrics
```

### Docker

https://hub.docker.com/r/czerwonk/ping_exporter

To run the ping_exporter as a Docker container, run:

```bash
docker run -p 9427:9427 --name ping_exporter czerwonk/ping_exporter
```


## Contribute

Simply fork and create a pull-request. We'll try to respond in a timely fashion.

## License

MIT License, Copyright (c) 2018  
Philip Berndroth [pberndro](https://twitter.com/pberndro)  
Daniel Czerwonk [dan_nrw](https://twitter.com/dan_nrw)
