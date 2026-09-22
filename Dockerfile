FROM golang:1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder
WORKDIR /go/ping_exporter

# Download go modules and take advantage of docker build cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/ping_exporter


FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
ENV CONFIG_FILE="/config/config.yml"
ENV CMD_FLAGS=""
RUN apk --no-cache add ca-certificates libcap

WORKDIR /app
COPY --from=builder /go/bin/ping_exporter .
RUN setcap cap_net_raw+ep /app/ping_exporter

CMD ./ping_exporter --config.path $CONFIG_FILE $CMD_FLAGS
EXPOSE 9427
