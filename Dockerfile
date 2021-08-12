FROM golang as builder
ADD . /go/ping_exporter/
WORKDIR /go/ping_exporter
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/ping_exporter

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /go/bin/ping_exporter .

ENTRYPOINT ["./ping_exporter"]

EXPOSE 9427
