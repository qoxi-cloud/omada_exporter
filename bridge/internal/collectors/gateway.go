package collector

import (
	"fmt"
	"strconv"

	"github.com/RCooLeR/omada_exporter/internal/model"
	"github.com/RCooLeR/omada_exporter/internal/speedtest"
	"github.com/prometheus/client_golang/prometheus"
)

// collectSpeedtest emits the last speed test through a WAN, if one ran.
func (c *DeviceCollector) collectSpeedtest(ch chan<- prometheus.Metric, wanName string, wanLabels []string) {
	res, ok := speedtest.Lookup(wanName)
	if !ok {
		return
	}
	labels := append(append([]string{}, wanLabels...), res.SourceIP)
	ch <- prometheus.MustNewConstMetric(c.omadaWanSpeedtestDownload, prometheus.GaugeValue, res.DownloadMbps, labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaWanSpeedtestUpload, prometheus.GaugeValue, res.UploadMbps, labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaWanSpeedtestLatency, prometheus.GaugeValue, res.LatencyMs, labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaWanSpeedtestJitter, prometheus.GaugeValue, res.JitterMs, labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaWanSpeedtestLoss, prometheus.GaugeValue, res.PacketLossPct, labels...)
	runLabels := append(labels, res.ServerName, res.ServerLocation, res.ISP, res.ResultURL, res.Status(), res.Err)
	ch <- prometheus.MustNewConstMetric(c.omadaWanSpeedtestLastRun, prometheus.GaugeValue, float64(res.RanAt.Unix()), runLabels...)
}

// collectGateway emits metrics for the gateway.
func (c *DeviceCollector) collectGateway(ch chan<- prometheus.Metric, gateway *model.Gateway, siteID string) error {
	labels := []string{
		gateway.GetMac(),
		gateway.GetType(),
		gateway.GetSubtype(),
		gateway.GetModel(),
		gateway.GetShowModel(),
		gateway.GetVersion(),
		gateway.GetVersionWithUpgrade(),
		gateway.GetHwVersion(),
		gateway.GetFirmwareVersion(),
		gateway.GetIp(),
		gateway.GetName(),
		gateway.GetStatus(),
		c.client.Settings().Site,
		siteID,
	}
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceTemp, prometheus.GaugeValue, gateway.GetTemp(), labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceTxRate, prometheus.GaugeValue, gateway.TxRate, labels...)
	ch <- prometheus.MustNewConstMetric(c.omadaDeviceRxRate, prometheus.GaugeValue, gateway.RxRate, labels...)
	for _, wan := range gateway.Wans {
		wanLabels := append(labels,
			fmt.Sprintf("%d", wan.Port),
			wan.Name,
			wan.Desc,
			wan.GetType(),
			wan.Ip,
			wan.Proto,
		)
		ch <- prometheus.MustNewConstMetric(c.omadaWanStatus, prometheus.GaugeValue, float64(wan.Status), wanLabels...)
		ch <- prometheus.MustNewConstMetric(c.omadaWanInternetState, prometheus.GaugeValue, float64(wan.InternetState), wanLabels...)
		ch <- prometheus.MustNewConstMetric(c.omadaWanLinkSpeedMbps, prometheus.GaugeValue, float64(wan.GetLinkSpeed()), wanLabels...)
		ch <- prometheus.MustNewConstMetric(c.omadaWanRxRate, prometheus.GaugeValue, wan.RxRate, wanLabels...)
		ch <- prometheus.MustNewConstMetric(c.omadaWanTxRate, prometheus.GaugeValue, wan.TxRate, wanLabels...)
		ch <- prometheus.MustNewConstMetric(c.omadaWanLatency, prometheus.GaugeValue, float64(wan.Latency), wanLabels...)
		c.collectSpeedtest(ch, wan.Name, wanLabels)
	}
	if c.trackPortMetrics() {
		for _, port := range gateway.Ports {
			portLabels := c.buildPortLabels(
				labels,
				fmt.Sprintf("%d", port.Port),
				fmt.Sprintf("%d", port.MaxSpeed),
				port.Name,
				port.GetType(),
				port.Operation,
				port.GetLinkStatus(),
				fmt.Sprintf("%d", port.GetLinkSpeed()),
				strconv.FormatBool(port.Poe),
				port.GetLinkSpeedLabel(),
			)
			ch <- prometheus.MustNewConstMetric(c.omadaPortLinkStatus, prometheus.GaugeValue, float64(port.LinkStatus), portLabels...)
			if port.PoePower > 0 {
				ch <- prometheus.MustNewConstMetric(c.omadaPortPowerWatts, prometheus.GaugeValue, port.PoePower, portLabels...)
			}
			ch <- prometheus.MustNewConstMetric(c.omadaPortLinkSpeedMbps, prometheus.GaugeValue, float64(port.GetLinkSpeed()), portLabels...)
			ch <- prometheus.MustNewConstMetric(c.omadaPortLinkRx, prometheus.CounterValue, port.Rx, portLabels...)
			ch <- prometheus.MustNewConstMetric(c.omadaPortLinkTx, prometheus.CounterValue, port.Tx, portLabels...)
		}
	}
	return nil
}
