package omada

import (
	"testing"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func TestRttMsUsesBestRoundTripAndZeroWhenSilent(t *testing.T) {
	if got := rttMs(nil); got != 0 {
		t.Fatalf("nil stats = %v, want 0", got)
	}
	if got := rttMs(&probing.Statistics{PacketsSent: 5}); got != 0 {
		t.Fatalf("no replies = %v, want 0", got)
	}
	// a control-plane burst: four slow answers hide a 2 ms link
	st := &probing.Statistics{
		PacketsSent: 5,
		PacketsRecv: 5,
		MinRtt:      2100 * time.Microsecond,
		AvgRtt:      62 * time.Millisecond,
		MaxRtt:      79 * time.Millisecond,
	}
	if got := rttMs(st); got != 2.1 {
		t.Fatalf("best rtt = %v ms, want 2.1", got)
	}
}

func TestPingGatewayMsSkipsEmptyGateway(t *testing.T) {
	for _, ip := range []string{"", "0.0.0.0"} {
		if got := pingGatewayMs(ip); got != 0 {
			t.Fatalf("pingGatewayMs(%q) = %v, want 0", ip, got)
		}
	}
}
