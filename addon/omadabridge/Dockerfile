ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:3.21
ARG FORK_REPO=https://github.com/qoxi-cloud/omada_exporter
# Supervisor passes BUILD_VERSION = version from config.yaml; the bridge source
# is checked out at the git tag of the same name (v<version>), so an add-on
# release is exactly one tag in this repository. FORK_REF overrides it.
ARG BUILD_VERSION
ARG FORK_REF=

FROM golang:1.27-alpine AS builder
ARG FORK_REPO
ARG FORK_REF
ARG BUILD_VERSION
RUN apk add --no-cache git ca-certificates
WORKDIR /src
RUN ref="${FORK_REF:-v${BUILD_VERSION}}" \
 && git clone --quiet "${FORK_REPO}" . \
 && git checkout --quiet "${ref}" \
 && echo "building bridge at ${ref}"
WORKDIR /src/bridge
ENV CGO_ENABLED=0 GOFLAGS=-buildvcs=false
RUN go build -trimpath -ldflags="-s -w" -o /omada-exporter .

FROM ${BUILD_FROM}
COPY --from=builder /omada-exporter /usr/bin/omada-exporter
COPY run.sh /run.sh
RUN chmod a+x /run.sh /usr/bin/omada-exporter
HEALTHCHECK --interval=60s --timeout=5s --start-period=30s \
  CMD wget -q -T 5 -O /dev/null http://127.0.0.1:9202/healthz || exit 1
CMD ["/run.sh"]
