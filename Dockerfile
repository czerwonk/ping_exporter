FROM golang:1.17.5-alpine3.15 as builder
ADD . /go/ping_exporter/
WORKDIR /go/ping_exporter
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/ping_exporter


FROM alpine:latest
ENV CONFIG_FILE "/config/config.yml"
ENV CMD_FLAGS ""

WORKDIR /app
COPY --from=builder /go/bin/ping_exporter .
RUN apk --no-cache add ca-certificates libcap && \
    setcap cap_net_raw+ep /app/ping_exporter

CMD ./ping_exporter --config.path $CONFIG_FILE $CMD_FLAGS
EXPOSE 9427
