package omada

import (
	"os"
	"strings"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/rs/zerolog/log"

	"github.com/RCooLeR/omada_exporter/internal/model"
)

// virtualWanPingEnabled reads OMADA_VWAN_PING (default true). Omada reports no
// latency for virtual WANs, so the bridge pings each virtual WAN's gateway
// itself; the gateway sits in that WAN's connected subnet, so the probe leaves
// through the right uplink regardless of policy routing.
func virtualWanPingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("OMADA_VWAN_PING")))
	return v == "" || v == "1" || v == "true" || v == "yes"
}

const (
	vwanPingCount    = 3
	vwanPingInterval = 200 * time.Millisecond
	vwanPingTimeout  = 2 * time.Second
)

// pingGatewayMs returns the average RTT in milliseconds to ip, or 0 when the
// host does not answer or pinging is impossible (no raw-socket permission).
func pingGatewayMs(ip string) float64 {
	if ip == "" || ip == "0.0.0.0" {
		return 0
	}
	run := func(privileged bool) (float64, error) {
		p, err := probing.NewPinger(ip)
		if err != nil {
			return 0, err
		}
		p.SetPrivileged(privileged)
		p.Count = vwanPingCount
		p.Interval = vwanPingInterval
		p.Timeout = vwanPingTimeout
		if err := p.Run(); err != nil {
			return 0, err
		}
		st := p.Statistics()
		if st == nil || st.PacketsRecv == 0 {
			return 0, nil
		}
		return float64(st.AvgRtt) / float64(time.Millisecond), nil
	}
	ms, err := run(true)
	if err != nil {
		ms, err = run(false)
	}
	if err != nil {
		log.Warn().Err(err).Str("ip", ip).Msg("virtual WAN gateway ping failed")
		return 0
	}
	return ms
}

// pingVirtualWans probes every virtual WAN gateway concurrently and returns
// name → average RTT (ms).
func pingVirtualWans(vws []model.VirtualWan) map[string]float64 {
	out := map[string]float64{}
	if !virtualWanPingEnabled() || len(vws) == 0 {
		return out
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, vw := range vws {
		wg.Add(1)
		go func(vw model.VirtualWan) {
			defer wg.Done()
			ms := pingGatewayMs(vw.Ipv4.Gateway)
			mu.Lock()
			out[vw.Name] = ms
			mu.Unlock()
		}(vw)
	}
	wg.Wait()
	return out
}
