package speedtest

import (
	"math"
	"os"
	"testing"
	"time"
)

func TestParseSources(t *testing.T) {
	got, err := ParseSources(" LinkCom_sfp2=172.21.9.202, Kopiyka_sfp3=172.21.9.203 ,,")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != (Source{"LinkCom_sfp2", "172.21.9.202"}) || got[1] != (Source{"Kopiyka_sfp3", "172.21.9.203"}) {
		t.Fatalf("sources = %+v", got)
	}
	if _, err := ParseSources("broken"); err == nil {
		t.Fatal("expected error for entry without =")
	}
}

func TestParseOutputResult(t *testing.T) {
	raw, err := os.ReadFile("testdata/ookla.json")
	if err != nil {
		t.Fatal(err)
	}
	src := Source{WAN: "LinkCom_sfp2", IP: "172.21.9.202"}
	res, err := ParseOutput(src, raw, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	if !near(res.DownloadMbps, 937.72) || !near(res.UploadMbps, 469.71) {
		t.Fatalf("bandwidth = %.2f / %.2f", res.DownloadMbps, res.UploadMbps)
	}
	if res.LatencyMs != 3.871 || res.JitterMs != 0.412 || res.PacketLossPct != 0 {
		t.Fatalf("ping = %+v", res)
	}
	if res.ServerName != "LinkCom" || res.ServerLocation != "Kyiv, Ukraine" || res.ISP != "LinkCom" || res.ResultURL != "https://www.speedtest.net/result/c/abc" {
		t.Fatalf("server = %+v", res)
	}
	if res.RanAt.Format(time.RFC3339) != "2026-10-28T22:40:33Z" || res.Status() != "ok" || res.SourceIP != src.IP {
		t.Fatalf("meta = %+v", res)
	}
}

func TestParseOutputError(t *testing.T) {
	raw, _ := os.ReadFile("testdata/ookla-error.json")
	_, err := ParseOutput(Source{"x", "1.2.3.4"}, raw, time.Now())
	if err == nil || err.Error() != "Cannot open socket: Cannot assign requested address" {
		t.Fatalf("err = %v", err)
	}
	if _, err := ParseOutput(Source{"x", "1.2.3.4"}, nil, time.Now()); err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestLookupWithoutRunner(t *testing.T) {
	Install(nil)
	if _, ok := Lookup("anything"); ok {
		t.Fatal("lookup must miss when no runner is installed")
	}
	r := New("", time.Hour, nil)
	r.results["linkcom_sfp2"] = Result{WAN: "LinkCom_sfp2", DownloadMbps: 1}
	Install(r)
	if res, ok := Lookup("LINKCOM_SFP2"); !ok || res.DownloadMbps != 1 {
		t.Fatalf("lookup = %+v %v", res, ok)
	}
}
