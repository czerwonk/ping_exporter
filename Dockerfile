FROM golang

ENV CONFIG_FILE "/config"

RUN apt-get install -y git && \
    go get github.com/czerwonk/ping_exporter

CMD ping_exporter -config.path $CONFIG_FILE
EXPOSE 9427
