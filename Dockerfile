FROM golang as builder
RUN go get -d -v github.com/czerwonk/ping_exporter
WORKDIR /go/src/github.com/czerwonk/ping_exporter
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
ENV CONFIG_FILE "/config"
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /go/src/github.com/czerwonk/ping_exporter/app ping_exporter
CMD ./ping_exporter --config.path $CONFIG_FILE
EXPOSE 9427
