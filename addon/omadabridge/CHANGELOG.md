## 2.4.4-qoxi.16

- Per-client sensors get `default_entity_id: sensor.net_client_<mac>_<metric>` (same HA 2026.4 `object_id` removal as the trackers). Name-based ids collided between clients with equal names (`sensor.lwip0_client_rssi_dbm_2` … `_8`) and drifted on rename. Site aggregates (`net_client_connected_total`) and all infrastructure entities keep their name-based ids.
- Per-client sensors no longer send `state_class`: they are kept out of the recorder, and HA flagged each of them as `entity_not_recorded` in statistics validation.
- Control buttons send `default_entity_id: button.<control id>` instead of the ignored `object_id`.
- Device trackers send `name: null`, so the friendly name is the device name once (was "iPhone iPhone").
- `value` is stripped from HA attributes as well: it duplicates the state, so every state change also wrote a new `state_attributes` row.

## 2.4.4-qoxi.15

- Client device trackers now get `default_entity_id: device_tracker.net_client_<mac>`. Home Assistant 2026.4 removed the MQTT `object_id` key (dropped silently), so trackers were being named `device_tracker.<name>_<name>` and could not be matched by a recorder glob. The ignored `object_id` is no longer sent for trackers. Only affects first registration; existing entities keep their registry id.

## 2.4.4-qoxi.14

- Home Assistant attributes no longer carry the per-poll `last_updated` timestamp (`json_attributes_template` strips it; the state payload keeps it for client retention). Without this every bridge entity produced a new recorder row every poll — ~1.3M rows / 1.7 GB per half day on PostgreSQL.

## 2.4.4-qoxi.13
- Virtual WAN latency: 5 probes 400 ms apart, the WAN reports the best RTT (not the average), and a fully lost burst is retried once. ISP gateways answer ICMP from their control plane: one uplink showed 45–80 ms every ~3 min and occasional silent bursts while the link itself sat at 2 ms, which flipped the dashboard tile orange and to "—".

## 2.4.4-qoxi.12
- Back to the qoxi.9 footprint: no `host_network`, `NET_RAW` only, no iproute2. Nothing of the speed test remains.

## 2.4.4-qoxi.11
- Speed test removed (reverted qoxi.10: no Ookla CLI, no `speedtest_*` options, no `net_wan_speedtest_*` metrics). This release still runs with `host_network` + `NET_ADMIN` once, to remove the host aliases 172.21.9.202–204 that qoxi.10 added.

## 2.4.4-qoxi.10
- Per-WAN speed test: the add-on binds the Ookla CLI to one source address per WAN (`speedtest_sources`, aliases added on the host interface at start) and publishes `net_wan_speedtest_download_mbps`, `_upload_mbps`, `_latency_ms`, `_jitter_ms`, `_packet_loss_percentage`, `_last_run_timestamp` every `speedtest_interval_minutes` (default 60). Requires `host_network` + `NET_ADMIN` and one policy-routing rule per source address on the gateway.

## 2.4.4-qoxi.9
- Removed the one-time startup purge of `omada_*` retained records (migration helper, no longer needed). Client retention unchanged.

## 2.4.4-qoxi.8
- Add-on now lives in the bridge repository (`addon/omadabridge`); the image builds the bridge at git tag `v<version>`, so one tag = one release. Repository `qoxi-cloud/omadabridge-addon` is archived.

## 2.4.4-qoxi.7
- Retention: trackers restored from MQTT without a `last_seen` now start their retention window at restart instead of living forever.

## 2.4.4-qoxi.6
- Fix: the legacy `omada_*` purge in qoxi.5 matched the wrong prefix and removed nothing.

## 2.4.4-qoxi.5
- **Breaking, one-time:** all metrics are now vendor-neutral `net_*` (was `omada_*`), entity unique ids are `net_bridge_*` / `net_client_*`. On first start the bridge removes every retained `omada_*` entity and re-announces everything, so Home Assistant recreates the entities (same entity ids, history restarts).
- New option `client_retention_hours` (default 168): sensors and device trackers of clients not seen for that long are removed instead of lingering as unavailable. `0` keeps them forever.

## 2.4.4-qoxi.4
- Clients: `network_name`, `health_score`, `uptime_seconds`, `last_seen`, `channel`, `radio_id`, `power_save`, `guest`, `blocked` attributes.
- Devices: new sensor `Device Health Score` (0-10) with `client_health_good/fair/poor/no_data`, `device_serial`, `device_fan_status`, `device_loopback_num`, `device_block_num`.
- Switches: PoE budget read from Omada 6.x `poeTotalPower` (was 0).
- APs: `device_wpXg_channel`, `device_wpXg_tx_power`, `device_wpXg_clients` per band.

## 2.4.4-qoxi.3
- Wired uplink of every switch/AP from the controller topology map: new sensor `Device Uplink Link Speed Mbps` with attributes `uplink_name`, `uplink_mac`, `uplink_port`, `uplink_local_port`, `uplink_lag_id`, `uplink_lag_ports` (used by the Mushroom fork cards for client paths and switch port labels).

## 2.4.4-qoxi.2
- Latency for virtual WANs: the bridge pings each virtual WAN gateway (OMADA_VWAN_PING, default on); add-on gets NET_RAW.

## 2.4.4-qoxi.1
- Build from the qoxi-cloud fork (https://github.com/qoxi-cloud/omada_exporter, branch `qoxi`).
- Virtual WANs (Omada 6.x) are exported as WAN/ISP rows; physical ports that only carry virtual WANs are hidden.

## 2.4.4
- Initial add-on packaging of upstream OmadaBridge v2.4.4.
