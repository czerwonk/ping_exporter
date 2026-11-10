FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

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


FROM gcr.io/distroless/static-debian13@sha256:448810357cc8e67b7208988abdba0f5ec3d3aebee0a5b13eb624979792cffc0f
ENV CONFIG_FILE="/config/config.yml"
ENV CMD_FLAGS=""

WORKDIR /app
COPY --from=builder /out/ping_exporter /app/ping_exporter

ENTRYPOINT ["/app/ping_exporter"]
EXPOSE 9427
