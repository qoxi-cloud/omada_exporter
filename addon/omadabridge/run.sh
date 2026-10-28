#!/usr/bin/with-contenv bashio
# Map add-on options → OmadaBridge environment, then exec the upstream binary.
set -e

export OMADA_HOST="$(bashio::config 'omada_host')"
export OMADA_SITE="$(bashio::config 'omada_site')"
export OMADA_USER="$(bashio::config 'omada_user')"
export OMADA_PASS="$(bashio::config 'omada_pass')"
export OMADA_SYSTEM_TYPE="auto"
export OMADA_OPENAPI_AUTH="auto"
export OMADA_PORT="9202"
export LOG_LEVEL="$(bashio::config 'log_level')"

if bashio::config.true 'omada_insecure'; then export OMADA_INSECURE="true"; else export OMADA_INSECURE="false"; fi

if bashio::config.has_value 'omada_client_id' && bashio::config.has_value 'omada_secret_id'; then
  export OMADA_CLIENT_ID="$(bashio::config 'omada_client_id')"
  export OMADA_SECRET_ID="$(bashio::config 'omada_secret_id')"
else
  bashio::log.warning "No Open API client id/secret: WAN, ISP, VPN and client detail metrics will be skipped"
fi

for k in port_metrics client_metrics network_quality_metrics insight_metrics; do
  var="OMADA_TRACK_$(echo "$k" | tr '[:lower:]' '[:upper:]')"
  if bashio::config.true "track_${k}"; then export "$var=true"; else export "$var=false"; fi
done

# MQTT: explicit options win; otherwise use the broker the Supervisor advertises (Mosquitto add-on)
export OMADA_MQTT_ENABLED="true"
export OMADA_MQTT_CLIENT_ID="omadabridge"
export OMADA_MQTT_TOPIC_PREFIX="omada_exporter"
export OMADA_MQTT_DISCOVERY_PREFIX="homeassistant"
export OMADA_MQTT_RETAIN="true"
export OMADA_MQTT_INTERVAL="$(bashio::config 'mqtt_interval')"
export OMADA_MQTT_EXPIRE_AFTER="$(bashio::config 'mqtt_expire_after')"
export OMADA_MQTT_CLIENT_RETENTION_HOURS="$(bashio::config 'client_retention_hours')"

if bashio::config.has_value 'mqtt_host'; then
  export OMADA_MQTT_BROKER="tcp://$(bashio::config 'mqtt_host'):$(bashio::config 'mqtt_port')"
  bashio::config.has_value 'mqtt_user' && export OMADA_MQTT_USER="$(bashio::config 'mqtt_user')"
  bashio::config.has_value 'mqtt_pass' && export OMADA_MQTT_PASS="$(bashio::config 'mqtt_pass')"
elif bashio::services.available 'mqtt'; then
  export OMADA_MQTT_BROKER="tcp://$(bashio::services 'mqtt' 'host'):$(bashio::services 'mqtt' 'port')"
  export OMADA_MQTT_USER="$(bashio::services 'mqtt' 'username')"
  export OMADA_MQTT_PASS="$(bashio::services 'mqtt' 'password')"
  bashio::log.info "Using Supervisor-provided MQTT broker ${OMADA_MQTT_BROKER}"
else
  bashio::exit.nok "No MQTT broker: install the Mosquitto add-on or set mqtt_host/mqtt_user/mqtt_pass"
fi

# Speed test: one source address per WAN; the gateway policy-routes each
# address to its WAN, so the Ookla CLI bound to that address measures one ISP.
export OMADA_SPEEDTEST_INTERVAL_MINUTES="$(bashio::config 'speedtest_interval_minutes')"
export OMADA_SPEEDTEST_BIN="/usr/bin/speedtest"
sources=""
if bashio::config.has_value 'speedtest_sources'; then
  iface="$(ip -4 route get 1.1.1.1 2>/dev/null | sed -n 's/.* dev \([^ ]*\).*/\1/p' | head -1)"
  prefix="$(ip -4 -o addr show dev "${iface}" scope global 2>/dev/null | head -1 | sed -n 's|.* inet [0-9.]*/\([0-9]*\) .*|\1|p')"
  for key in $(bashio::config 'speedtest_sources|keys'); do
    wan="$(bashio::config "speedtest_sources[${key}].wan")"
    ip_addr="$(bashio::config "speedtest_sources[${key}].ip")"
    sources="${sources:+${sources},}${wan}=${ip_addr}"
    if [ -z "${iface}" ] || [ -z "${prefix}" ]; then
      bashio::log.warning "speedtest: cannot determine host interface/prefix, not adding alias ${ip_addr}"
    elif ip -4 -o addr show dev "${iface}" | grep -q " ${ip_addr}/"; then
      bashio::log.info "speedtest: alias ${ip_addr} already on ${iface}"
    elif ip addr add "${ip_addr}/${prefix}" dev "${iface}" 2>/dev/null; then
      bashio::log.info "speedtest: added alias ${ip_addr}/${prefix} on ${iface} for ${wan}"
    else
      bashio::log.warning "speedtest: failed to add alias ${ip_addr}/${prefix} on ${iface} (NET_ADMIN + host_network required)"
    fi
  done
fi
export OMADA_SPEEDTEST_SOURCES="${sources}"

# never enable device controls from this add-on: monitoring only
unset OMADA_ALLOW_DEVICE_CONTROL

bashio::log.info "Starting OmadaBridge against ${OMADA_HOST} site '${OMADA_SITE}'"
exec /usr/bin/omada-exporter
