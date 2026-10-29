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

// ISP gateways answer ICMP from their control plane, not the forwarding path:
// one of the uplinks here sits at 2 ms but every ~3 minutes a whole burst comes
// back at 45–80 ms, and now and then a burst is dropped entirely. The link
// latency is therefore the *minimum* of a short, spaced burst, and a fully
// lost burst is retried once before the WAN is reported as not answering.
const (
	vwanPingCount    = 5
	vwanPingInterval = 400 * time.Millisecond
	vwanPingTimeout  = 4 * time.Second
	vwanPingRetries  = 1
	vwanRetryPause   = 500 * time.Millisecond
)

// rttMs picks the latency a burst stands for: the best round trip in
// milliseconds, or 0 when nothing came back.
func rttMs(st *probing.Statistics) float64 {
	if st == nil || st.PacketsRecv == 0 {
		return 0
	}
	return float64(st.MinRtt) / float64(time.Millisecond)
}

// pingGatewayMs returns the best RTT in milliseconds to ip, or 0 when the host
// does not answer or pinging is impossible (no raw-socket permission).
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
		return rttMs(p.Statistics()), nil
	}
	privileged := true
	var ms float64
	for attempt := 0; attempt <= vwanPingRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(vwanRetryPause)
		}
		var err error
		ms, err = run(privileged)
		if err != nil && privileged {
			privileged = false
			ms, err = run(privileged)
		}
		if err != nil {
			log.Warn().Err(err).Str("ip", ip).Msg("virtual WAN gateway ping failed")
			return 0
		}
		if ms > 0 {
			return ms
		}
	}
	log.Debug().Str("ip", ip).Msg("virtual WAN gateway did not answer")
	return 0
}

// pingVirtualWans probes every virtual WAN gateway concurrently and returns
// name → best RTT (ms).
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
