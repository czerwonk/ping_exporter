FROM golang as builder
ADD . /go/ping_exporter/
WORKDIR /go/ping_exporter
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/ping_exporter

FROM alpine:latest
ENV CONFIG_FILE "/config"
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /go/bin/ping_exporter .
CMD ./ping_exporter --config.path $CONFIG_FILE --web.listen-address 0.0.0.0:9247
EXPOSE 9427
