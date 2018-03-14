FROM golang

RUN apt-get install -y git && \
    go get github.com/czerwonk/ping_exporter

CMD junos_exporter $TARGETS
EXPOSE 9427