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
ARG BUILD_ARCH=amd64
ARG SPEEDTEST_VERSION=1.2.0
# Ookla CLI is a static musl build; needed for the per-WAN speed test.
# iproute2 adds the per-WAN source addresses on the host interface at start.
RUN apk add --no-cache iproute2 \
 && case "${BUILD_ARCH}" in \
      amd64) st_arch=x86_64 ;; \
      aarch64) st_arch=aarch64 ;; \
      *) echo "unsupported arch ${BUILD_ARCH}" && exit 1 ;; \
    esac \
 && wget -q -O /tmp/speedtest.tgz "https://install.speedtest.net/app/cli/ookla-speedtest-${SPEEDTEST_VERSION}-linux-${st_arch}.tgz" \
 && tar -xzf /tmp/speedtest.tgz -C /usr/bin speedtest \
 && rm -f /tmp/speedtest.tgz \
 && /usr/bin/speedtest --version | head -1
COPY --from=builder /omada-exporter /usr/bin/omada-exporter
COPY run.sh /run.sh
RUN chmod a+x /run.sh /usr/bin/omada-exporter /usr/bin/speedtest
HEALTHCHECK --interval=60s --timeout=5s --start-period=30s \
  CMD wget -q -T 5 -O /dev/null http://127.0.0.1:9202/healthz || exit 1
CMD ["/run.sh"]
