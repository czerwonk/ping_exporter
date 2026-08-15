FROM golang:1.26.6-alpine3.24@sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df AS builder
WORKDIR /go/ping_exporter

# Download go modules and take advantage of docker build cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/ping_exporter


FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
ENV CONFIG_FILE="/config/config.yml"
ENV CMD_FLAGS=""
RUN apk --no-cache add ca-certificates libcap

WORKDIR /app
COPY --from=builder /go/bin/ping_exporter .
RUN setcap cap_net_raw+ep /app/ping_exporter

CMD ./ping_exporter --config.path $CONFIG_FILE $CMD_FLAGS
EXPOSE 9427
