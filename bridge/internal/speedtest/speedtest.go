// Package speedtest runs the Ookla CLI periodically, once per configured
// source address, and keeps the last result per WAN in memory for the
// collectors. Each source address is policy-routed by the gateway to one WAN,
// which is how a LAN-side host measures one ISP at a time.
package speedtest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DefaultBinary   = "/usr/bin/speedtest"
	firstRunDelay   = 90 * time.Second
	runTimeout      = 180 * time.Second
	bitsPerMegabit  = 1e6
	bytesToMegabits = 8 / bitsPerMegabit
)

// Source maps a WAN name (the `name` label of net_wan_* metrics) to the local
// address a test must bind to.
type Source struct {
	WAN string
	IP  string
}

// Result is the last outcome for one source. Error results keep the previous
// numbers at zero so a failed hour is visible rather than silently stale.
type Result struct {
	WAN            string
	SourceIP       string
	DownloadMbps   float64
	UploadMbps     float64
	LatencyMs      float64
	JitterMs       float64
	PacketLossPct  float64
	ServerName     string
	ServerLocation string
	ISP            string
	ResultURL      string
	RanAt          time.Time
	Err            string
}

// Status is "ok" or "error", as published in the speedtest_status label.
func (r Result) Status() string {
	if r.Err != "" {
		return "error"
	}
	return "ok"
}

// ParseSources parses "WAN=ip,WAN2=ip2"; blank entries are skipped.
func ParseSources(spec string) ([]Source, error) {
	var out []Source
	for _, item := range strings.Split(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		wan, ip, ok := strings.Cut(item, "=")
		wan, ip = strings.TrimSpace(wan), strings.TrimSpace(ip)
		if !ok || wan == "" || ip == "" {
			return nil, fmt.Errorf("speedtest source %q must be WAN=ip", item)
		}
		out = append(out, Source{WAN: wan, IP: ip})
	}
	return out, nil
}

// ooklaOutput is the subset of `speedtest -f json` the bridge uses.
type ooklaOutput struct {
	Type       string  `json:"type"`
	Timestamp  string  `json:"timestamp"`
	ISP        string  `json:"isp"`
	Message    string  `json:"message"`
	Error      string  `json:"error"`
	PacketLoss float64 `json:"packetLoss"`
	Ping       struct {
		Latency float64 `json:"latency"`
		Jitter  float64 `json:"jitter"`
	} `json:"ping"`
	Download struct {
		Bandwidth float64 `json:"bandwidth"`
	} `json:"download"`
	Upload struct {
		Bandwidth float64 `json:"bandwidth"`
	} `json:"upload"`
	Server struct {
		Name     string `json:"name"`
		Location string `json:"location"`
		Country  string `json:"country"`
	} `json:"server"`
	Result struct {
		URL string `json:"url"`
	} `json:"result"`
}

// ParseOutput turns the CLI JSON (possibly several lines: progress lines
// followed by the final "result" line) into a Result. Bandwidths are bytes/s.
func ParseOutput(src Source, raw []byte, ranAt time.Time) (Result, error) {
	res := Result{WAN: src.WAN, SourceIP: src.IP, RanAt: ranAt}
	var final *ooklaOutput
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var out ooklaOutput
		if err := json.Unmarshal(line, &out); err != nil {
			continue
		}
		if out.Type == "result" || out.Type == "log" && out.Error != "" {
			o := out
			final = &o
		}
	}
	if final == nil {
		return res, errors.New("no result line in speedtest output")
	}
	if final.Type != "result" {
		msg := final.Error
		if msg == "" {
			msg = final.Message
		}
		return res, errors.New(msg)
	}
	res.DownloadMbps = final.Download.Bandwidth * bytesToMegabits
	res.UploadMbps = final.Upload.Bandwidth * bytesToMegabits
	res.LatencyMs = final.Ping.Latency
	res.JitterMs = final.Ping.Jitter
	res.PacketLossPct = final.PacketLoss
	res.ServerName = final.Server.Name
	res.ServerLocation = strings.TrimSpace(strings.Join([]string{final.Server.Location, final.Server.Country}, ", "))
	res.ServerLocation = strings.Trim(res.ServerLocation, ", ")
	res.ISP = final.ISP
	res.ResultURL = final.Result.URL
	if ts, err := time.Parse(time.RFC3339, final.Timestamp); err == nil {
		res.RanAt = ts
	}
	return res, nil
}

// Runner owns the schedule and the last results.
type Runner struct {
	binary   string
	interval time.Duration
	sources  []Source
	mu       sync.RWMutex
	results  map[string]Result
	runMu    sync.Mutex
}

var (
	globalMu sync.RWMutex
	global   *Runner
)

// New builds a runner; interval <= 0 or no sources means it never runs.
func New(binary string, interval time.Duration, sources []Source) *Runner {
	if binary == "" {
		binary = DefaultBinary
	}
	return &Runner{binary: binary, interval: interval, sources: sources, results: map[string]Result{}}
}

// Install makes r the process-wide source for collectors.
func Install(r *Runner) {
	globalMu.Lock()
	global = r
	globalMu.Unlock()
}

// Lookup returns the last result for a WAN name (case-insensitive).
func Lookup(wan string) (Result, bool) {
	globalMu.RLock()
	r := global
	globalMu.RUnlock()
	if r == nil {
		return Result{}, false
	}
	return r.Result(wan)
}

// Result returns the last result for a WAN name (case-insensitive).
func (r *Runner) Result(wan string) (Result, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.results[strings.ToLower(wan)]
	return res, ok
}

// Start schedules the loop; it returns immediately.
func (r *Runner) Start(ctx context.Context) {
	if r.interval <= 0 || len(r.sources) == 0 {
		log.Info().Msg("speedtest disabled (no interval or no sources)")
		return
	}
	log.Info().Dur("interval", r.interval).Int("sources", len(r.sources)).Str("binary", r.binary).Msg("speedtest scheduled")
	go r.loop(ctx)
}

func (r *Runner) loop(ctx context.Context) {
	timer := time.NewTimer(firstRunDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		r.RunAll(ctx)
		timer.Reset(r.interval)
	}
}

// RunAll tests every source one after another; tests never overlap because
// two concurrent measurements would halve each other's numbers.
func (r *Runner) RunAll(ctx context.Context) {
	r.runMu.Lock()
	defer r.runMu.Unlock()
	for _, src := range r.sources {
		if ctx.Err() != nil {
			return
		}
		res := r.runOne(ctx, src)
		r.mu.Lock()
		r.results[strings.ToLower(src.WAN)] = res
		r.mu.Unlock()
	}
}

func (r *Runner) runOne(ctx context.Context, src Source) Result {
	ranAt := time.Now().UTC()
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, r.binary, "--accept-license", "--accept-gdpr", "-f", "json", "-i", src.IP)
	raw, err := cmd.Output()
	res, perr := ParseOutput(src, raw, ranAt)
	switch {
	case perr == nil:
	case err != nil:
		res.Err = fmt.Sprintf("%v: %s", err, strings.TrimSpace(perr.Error()))
	default:
		res.Err = perr.Error()
	}
	ev := log.Info().Str("wan", src.WAN).Str("source_ip", src.IP)
	if res.Err != "" {
		ev.Str("error", res.Err).Msg("speedtest failed")
		return res
	}
	ev.Float64("down_mbps", res.DownloadMbps).Float64("up_mbps", res.UploadMbps).
		Float64("latency_ms", res.LatencyMs).Float64("jitter_ms", res.JitterMs).
		Float64("loss_pct", res.PacketLossPct).Str("server", res.ServerName).Msg("speedtest result")
	return res
}
