package collector

import (
	"fmt"

	"github.com/RCooLeR/omada_exporter/internal/model"
	"github.com/RCooLeR/omada_exporter/internal/omada"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/rs/zerolog/log"
)

// DeviceCollector collects and exports device metrics.
type DeviceCollector struct {
	omadaDeviceUptimeSeconds *prometheus.Desc
	omadaDeviceCpuPercentage *prometheus.Desc
	omadaDeviceMemPercentage *prometheus.Desc
	omadaDeviceNeedUpgrade   *prometheus.Desc
	omadaDeviceDownload      *prometheus.Desc
	omadaDeviceUpload        *prometheus.Desc
	omadaDeviceTemp          *prometheus.Desc
	omadaDeviceUplinkSpeed   *prometheus.Desc
	omadaDeviceHealthScore   *prometheus.Desc
	//ports
	omadaPortLinkStatus    *prometheus.Desc
	omadaPortPowerWatts    *prometheus.Desc
	omadaPortLinkSpeedMbps *prometheus.Desc
	omadaPortLinkRx        *prometheus.Desc
	omadaPortLinkTx        *prometheus.Desc
	//gateway
	omadaWanStatus        *prometheus.Desc
	omadaWanInternetState *prometheus.Desc
	omadaWanLinkSpeedMbps *prometheus.Desc
	omadaWanRxRate        *prometheus.Desc
	omadaWanTxRate        *prometheus.Desc
	omadaWanLatency       *prometheus.Desc
	//switch
	omadaDevicePoeRemainWatts *prometheus.Desc
	//lags
	omadaLagLinkStatus    *prometheus.Desc
	omadaLagLinkSpeedMbps *prometheus.Desc
	omadaLagLinkRx        *prometheus.Desc
	omadaLagLinkTx        *prometheus.Desc
	//ap
	omadaDeviceTxRate      *prometheus.Desc
	omadaDeviceRxRate      *prometheus.Desc
	omadaDevice2gTxUtil    *prometheus.Desc
	omadaDevice2gRxUtil    *prometheus.Desc
	omadaDevice5gTxUtil    *prometheus.Desc
	omadaDevice5gRxUtil    *prometheus.Desc
	omadaDevice5g2TxUtil   *prometheus.Desc
	omadaDevice5g2RxUtil   *prometheus.Desc
	omadaDevice6gTxUtil    *prometheus.Desc
	omadaDevice6gRxUtil    *prometheus.Desc
	omadaRadioInterference *prometheus.Desc

	client *omada.Client
}

// Describe sends the collector metric descriptors to Prometheus.
func (c *DeviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.omadaDeviceUptimeSeconds
	ch <- c.omadaDeviceCpuPercentage
	ch <- c.omadaDeviceMemPercentage
	ch <- c.omadaDeviceNeedUpgrade
	ch <- c.omadaDeviceDownload
	ch <- c.omadaDeviceUpload
	ch <- c.omadaDeviceTemp
	ch <- c.omadaDeviceUplinkSpeed
	ch <- c.omadaDeviceHealthScore
	if c.trackPortMetrics() {
		//ports
		ch <- c.omadaPortLinkStatus
		ch <- c.omadaPortPowerWatts
		ch <- c.omadaPortLinkSpeedMbps
		ch <- c.omadaPortLinkRx
		ch <- c.omadaPortLinkTx
	}
	//gateway
	ch <- c.omadaWanStatus
	ch <- c.omadaWanInternetState
	ch <- c.omadaWanLinkSpeedMbps
	ch <- c.omadaWanRxRate
	ch <- c.omadaWanTxRate
	ch <- c.omadaWanLatency
	//switch
	ch <- c.omadaDevicePoeRemainWatts
	//lags
	ch <- c.omadaLagLinkStatus
	ch <- c.omadaLagLinkSpeedMbps
	ch <- c.omadaLagLinkRx
	ch <- c.omadaLagLinkTx
	//ap
	ch <- c.omadaDeviceTxRate
	ch <- c.omadaDeviceRxRate
	ch <- c.omadaDevice2gTxUtil
	ch <- c.omadaDevice2gRxUtil
	ch <- c.omadaDevice5gTxUtil
	ch <- c.omadaDevice5gRxUtil
	ch <- c.omadaDevice5g2TxUtil
	ch <- c.omadaDevice5g2RxUtil
	ch <- c.omadaDevice6gTxUtil
	ch <- c.omadaDevice6gRxUtil
	ch <- c.omadaRadioInterference
}

// trackPortMetrics reports whether port metrics.
func (c *DeviceCollector) trackPortMetrics() bool {
	return trackPortMetrics(c.client)
}

// includePortActivityLabel reports whether port activity label.
func (c *DeviceCollector) includePortActivityLabel() bool {
	return includePortActivityLabel(c.client)
}

// buildPortLabels builds the port labels.
func (c *DeviceCollector) buildPortLabels(baseLabels []string, port, maxSpeed, name, portType, operation, linkStatus, linkSpeed, poe, activity string) []string {
	labels := append([]string{}, baseLabels...)
	labels = append(labels,
		port,
		maxSpeed,
		name,
		portType,
		operation,
		linkStatus,
		linkSpeed,
		poe,
	)
	if c.includePortActivityLabel() {
		labels = append(labels, activity)
	}
	return labels
}

// collectDevice emits metrics for the device.
func (c *DeviceCollector) collectDevice(ch chan<- prometheus.Metric, device model.DeviceInterface, siteID string) error {
	labels := []string{
		device.GetMac(),
		device.GetType(),
		device.GetSubtype(),
		device.GetModel(),
		device.GetShowModel(),
		device.GetVersion(),
		device.GetVersionWithUpgrade(),
		device.GetHwVersion(),
		device.GetFirmwareVersion(),
		device.GetIp(),
		device.GetName(),
		device.GetStatus(),
		c.client.Settings().Site,
		siteID,
	}
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceUptimeSeconds, prometheus.GaugeValue, device.GetUptime(), labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceCpuPercentage, prometheus.GaugeValue, device.GetCpuUtilization(), labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceMemPercentage, prometheus.GaugeValue, device.GetMemUtilization(), labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceNeedUpgrade, prometheus.GaugeValue, boolFloat64(device.GetNeedUpgrade()), labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceDownload, prometheus.CounterValue, device.GetDownload(), labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceUpload, prometheus.CounterValue, device.GetUpload(), labels...)
	health := device.GetClientHealth()
	if health == nil {
		health = &model.ClientHealth{}
	}
	fan, loops, blocks := "", "", ""
	if sw, ok := device.(*model.Switch); ok {
		fan = fmt.Sprintf("%d", sw.FanStatus)
		loops = fmt.Sprintf("%d", sw.LoopbackNum)
		blocks = fmt.Sprintf("%d", sw.BlockNum)
	}
	healthLabels := append(append([]string{}, labels...),
		fmt.Sprintf("%d", health.Good),
		fmt.Sprintf("%d", health.Fair),
		fmt.Sprintf("%d", health.Poor),
		fmt.Sprintf("%d", health.NoData),
		device.GetSerial(),
		fan,
		loops,
		blocks,
	)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceHealthScore, prometheus.GaugeValue, device.GetHealthScore(), healthLabels...)
	if up := device.GetTopology(); up != nil {
		uplinkLabels := append(append([]string{}, labels...),
			up.ParentMac,
			up.ParentName,
			up.ParentType,
			up.ParentPort,
			up.LocalPort,
			up.LagID,
			up.LagPorts,
		)
		ch <- prometheus.MustNewConstMetric(c.omadaDeviceUplinkSpeed, prometheus.GaugeValue, float64(up.LinkSpeedMbps), uplinkLabels...)
	}
	return nil
}

// Collect fetches current data and emits Prometheus metrics.
func (c *DeviceCollector) Collect(ch chan<- prometheus.Metric) {
	devices, err := c.client.GetDevices()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get devices")
		return
	}
	_, siteID := c.client.ContextIDs()

	for _, d := range devices {
		switch item := d.(type) {
		case *model.Gateway:
			err = c.collectDevice(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect device")
			}
			err = c.collectGateway(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect gateway")
			}
		case *model.Switch:
			err = c.collectDevice(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect device")
			}
			err = c.collectSwitch(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect switch")
			}
		case *model.AccessPoint:
			err = c.collectDevice(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect device")
			}
			err := c.collectAccessPoint(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect access point")
			}
		case *model.Olt:
			err = c.collectDevice(ch, item, siteID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect device")
			}
			err = c.collectOlt(ch, item)
			if err != nil {
				log.Error().Err(err).Msg("Failed to collect olt")
			}
		}
	}
}

// NewDeviceCollector builds the Prometheus descriptors used to export device,
// uplink, port, LAG, WAN, and radio metrics.
func NewDeviceCollector(apiClient *omada.Client) *DeviceCollector {
	deviceLabels := []string{
		"device_mac",
		"device_type",
		"device_subtype",
		"device_model",
		"device_show_model",
		"device_version",
		"device_version_upgrade",
		"device_hw_version",
		"device_firmware_version",
		"device_ip",
		"device_name",
		"device_status",
		"site",
		"site_id",
	}
	uplinkLabels := append(append([]string{}, deviceLabels...),
		"uplink_mac",
		"uplink_name",
		"uplink_type",
		"uplink_port",
		"uplink_local_port",
		"uplink_lag_id",
		"uplink_lag_ports",
	)
	healthLabels := append(append([]string{}, deviceLabels...),
		"client_health_good",
		"client_health_fair",
		"client_health_poor",
		"client_health_no_data",
		"device_serial",
		"device_fan_status",
		"device_loopback_num",
		"device_block_num",
	)
	gatewayWanLabels := append(deviceLabels,
		"port",
		"name",
		"desc",
		"type",
		"ip",
		"proto",
	)
	switchLabels := append(deviceLabels,
		"device_poe_support",
		"device_port_number",
		"device_total_power",
	)
	devicePortLabels := append(deviceLabels,
		"port",
		"max_speed",
		"name",
		"type",
		"operation",
		"link_status",
		"link_speed",
		"poe",
	)
	if includePortActivityLabel(apiClient) {
		devicePortLabels = append(devicePortLabels, "port_activity_label")
	}

	switchLagLabels := append(deviceLabels,
		"lag_id",
		"lag_type",
		"name",
		"link_status",
		"link_speed",
		"lag_ports",
	)

	apLabels := append(deviceLabels,
		"device_any_poe_enable",
		"device_wireless_linked",
		"device_wlan_group",

		"device_wp2g_mode",
		"device_wp2g_tx_max_rate",
		"device_wp2g_band_width",

		"device_wp5g_mode",
		"device_wp5g_tx_max_rate",
		"device_wp5g_band_width",

		"device_wp5g2_mode",
		"device_wp5g2_tx_max_rate",
		"device_wp5g2_band_width",

		"device_wp6g_mode",
		"device_wp6g_tx_max_rate",
		"device_wp6g_band_width",
		// qoxi
		"device_wp2g_channel",
		"device_wp2g_tx_power",
		"device_wp2g_clients",
		"device_wp5g_channel",
		"device_wp5g_tx_power",
		"device_wp5g_clients",
		"device_wp5g2_channel",
		"device_wp5g2_tx_power",
		"device_wp5g2_clients",
		"device_wp6g_channel",
		"device_wp6g_tx_power",
		"device_wp6g_clients",
	)

	return &DeviceCollector{
		omadaRadioInterference: prometheus.NewDesc("net_ap_interference_percentage",
			"Current radio interference utilization reported by Omada, in percent. Missing or invalid readings are omitted.",
			[]string{"device_mac", "device_name", "device_type", "site", "site_id", "radio_band"}, nil),
		omadaDeviceUptimeSeconds: prometheus.NewDesc("net_device_uptime_seconds",
			"Uptime of the device.",
			deviceLabels,
			nil,
		),
		omadaDeviceCpuPercentage: prometheus.NewDesc("net_device_cpu_percentage",
			"Percentage of device CPU used.",
			deviceLabels,
			nil,
		),
		omadaDeviceMemPercentage: prometheus.NewDesc("net_device_mem_percentage",
			"Percentage of device Memory used.",
			deviceLabels,
			nil,
		),
		omadaDeviceNeedUpgrade: prometheus.NewDesc("net_device_need_upgrade",
			"A boolean on whether the device needs an upgrade.",
			deviceLabels,
			nil,
		),
		omadaDeviceDownload: prometheus.NewDesc("net_device_download",
			"Device download traffic.",
			deviceLabels,
			nil,
		),
		omadaDeviceUpload: prometheus.NewDesc("net_device_upload",
			"Device upload traffic.",
			deviceLabels,
			nil,
		),
		omadaDeviceUplinkSpeed: prometheus.NewDesc("net_device_uplink_link_speed_mbps",
			"Negotiated speed of the device's wired uplink from the controller topology map; labels name the parent device and both port ends.",
			uplinkLabels,
			nil,
		),
		omadaDeviceHealthScore: prometheus.NewDesc("net_device_health_score",
			"Controller device health score (0-10); labels carry the good/fair/poor client split, serial, fan status and STP loop/block counts.",
			healthLabels,
			nil,
		),
		omadaDeviceTemp: prometheus.NewDesc("net_device_temp",
			"Device temperature.",
			deviceLabels,
			nil,
		),
		//ap & switch
		omadaDeviceTxRate: prometheus.NewDesc("net_device_tx_rate",
			"The tx rate of the device.",
			deviceLabels,
			nil,
		),
		omadaDeviceRxRate: prometheus.NewDesc("net_device_rx_rate",
			"The rx rate of the device.",
			deviceLabels,
			nil,
		),
		//ports
		omadaPortLinkStatus: prometheus.NewDesc("net_port_link_status",
			"A boolean representing the link status of the port.",
			devicePortLabels,
			nil,
		),
		omadaPortPowerWatts: prometheus.NewDesc("net_port_power_watts",
			"The current PoE usage of the port in watts.",
			devicePortLabels,
			nil,
		),
		omadaPortLinkSpeedMbps: prometheus.NewDesc("net_port_link_speed_mbps",
			"Port link speed in mbps. This is the capability of the connection, not the active throughput.",
			devicePortLabels,
			nil,
		),
		omadaPortLinkRx: prometheus.NewDesc("net_port_link_rx",
			"Bytes recieved on a port.",
			devicePortLabels,
			nil,
		),
		omadaPortLinkTx: prometheus.NewDesc("net_port_link_tx",
			"Bytes transmitted on a port.",
			devicePortLabels,
			nil,
		),
		//gateway
		omadaWanStatus: prometheus.NewDesc("net_wan_status",
			"The current status of the WAN connected/disconnected",
			gatewayWanLabels,
			nil,
		),
		omadaWanInternetState: prometheus.NewDesc("net_wan_internet_state",
			"The current status of the WAN internet state connected/disconnected",
			gatewayWanLabels,
			nil,
		),
		omadaWanLinkSpeedMbps: prometheus.NewDesc("net_wan_link_speed_mbps",
			"Wan link speed in mbps. This is the capability of the connection, not the active throughput.",
			gatewayWanLabels,
			nil,
		),
		omadaWanRxRate: prometheus.NewDesc("net_wan_rx_rate",
			"Wan RX rate (KB/s)",
			gatewayWanLabels,
			nil,
		),
		omadaWanTxRate: prometheus.NewDesc("net_wan_tx_rate",
			"Wan TX rate (KB/s)",
			gatewayWanLabels,
			nil,
		),
		omadaWanLatency: prometheus.NewDesc("net_wan_latency",
			"Wan latency (ms)",
			gatewayWanLabels,
			nil,
		),
		//switch
		omadaDevicePoeRemainWatts: prometheus.NewDesc("net_device_poe_remain_watts",
			"The remaining amount of PoE power for the device in watts.",
			switchLabels,
			nil,
		),
		//lag
		omadaLagLinkStatus: prometheus.NewDesc("net_lag_link_status",
			"A boolean representing the link status of the lag.",
			switchLagLabels,
			nil,
		),
		omadaLagLinkSpeedMbps: prometheus.NewDesc("net_lag_link_speed_mbps",
			"Lag link speed in mbps. This is the capability of the connection, not the active throughput.",
			switchLagLabels,
			nil,
		),
		omadaLagLinkRx: prometheus.NewDesc("net_lag_link_rx",
			"Bytes recieved on a lag.",
			switchLagLabels,
			nil,
		),
		omadaLagLinkTx: prometheus.NewDesc("net_lag_link_tx",
			"Bytes transmitted on a lag.",
			switchLagLabels,
			nil,
		),
		//ap
		omadaDevice2gTxUtil: prometheus.NewDesc("net_device_2g_tx_util",
			"The tx rate of the device on 2.4Ghz.",
			apLabels,
			nil,
		),
		omadaDevice2gRxUtil: prometheus.NewDesc("net_device_2g_rx_util",
			"The tx rate of the device on 2.4Ghz.",
			apLabels,
			nil,
		),
		omadaDevice5gTxUtil: prometheus.NewDesc("net_device_5g_tx_util",
			"The tx rate of the device on 5Ghz.",
			apLabels,
			nil,
		),
		omadaDevice5gRxUtil: prometheus.NewDesc("net_device_5g_rx_util",
			"The tx rate of the device on 5Ghz.",
			apLabels,
			nil,
		),
		omadaDevice5g2TxUtil: prometheus.NewDesc("net_device_5g2_tx_util",
			"The tx rate of the device on 5Ghz 2.",
			apLabels,
			nil,
		),
		omadaDevice5g2RxUtil: prometheus.NewDesc("net_device_5g2_rx_util",
			"The tx rate of the device on 5Ghz 2.",
			apLabels,
			nil,
		),
		omadaDevice6gTxUtil: prometheus.NewDesc("net_device_6g_tx_util",
			"The tx rate of the device on 6Ghz..",
			apLabels,
			nil,
		),
		omadaDevice6gRxUtil: prometheus.NewDesc("net_device_6g_rx_util",
			"The tx rate of the device on 6Ghz.",
			apLabels,
			nil,
		),
		//api-client
		client: apiClient,
	}
}
