FROM golang as builder
RUN go get github.com/czerwonk/ping_exporter

FROM alpine:latest

ENV CONFIG_FILE "/config"

RUN apk --no-cache add ca-certificates
RUN mkdir /app
WORKDIR /app
COPY --from=builder /go/bin/ping_exporter .

CMD ping_exporter -config.path $CONFIG_FILE
EXPOSE 9427
