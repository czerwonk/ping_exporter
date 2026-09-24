FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY *.go ./
COPY config ./config
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
  GOARM=${TARGETVARIANT#v} \
  go build -trimpath \
  -ldflags="-s -w" \
  -o /out/ping_exporter .

RUN apk --no-cache add libcap && \
  setcap cap_net_raw+ep /out/ping_exporter


FROM gcr.io/distroless/static-debian13@sha256:58133991db06659feaabe0f4e97a35cebf15ef4ea08f8a4c6d2ee5f75e4aa6a0
ENV CONFIG_FILE="/config/config.yml"
ENV CMD_FLAGS=""

WORKDIR /app
COPY --from=builder /out/ping_exporter /app/ping_exporter

ENTRYPOINT ["/app/ping_exporter"]
EXPOSE 9427
