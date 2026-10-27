FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24@sha256:d530cf60723abf1ffae0cd16bb546b379ec0550d2f3815e5b16ed69dbc00fba4 AS builder

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


FROM gcr.io/distroless/static-debian13@sha256:15f8637509792e05b97d83451b0b6ccfa4abd548f997e406ac1e72c87fe1fa78
ENV CONFIG_FILE="/config/config.yml"
ENV CMD_FLAGS=""

WORKDIR /app
COPY --from=builder /out/ping_exporter /app/ping_exporter

ENTRYPOINT ["/app/ping_exporter"]
EXPOSE 9427
