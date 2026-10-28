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
