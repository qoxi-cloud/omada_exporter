package model

// Radio represents radio data returned by Omada.
type Radio struct {
	RdMode           string   `json:"rdMode"`
	BandWidth        string   `json:"bandWidth"`
	MaxTxRate        int32    `json:"maxTxRate"`
	InterUtilization *float64 `json:"interUtil"`
	RxUtilization    float64  `json:"rxUtil"`
	TxUtilization    float64  `json:"txUtil"`
	// qoxi: "36  / 5180MHz" and the configured power in dBm
	ActualChannel string `json:"actualChannel"`
	TxPower       int32  `json:"txPower"`
}

// Channel returns just the channel number from Omada's "36  / 5180MHz" string.
func (r *Radio) Channel() string {
	if r == nil {
		return ""
	}
	text := r.ActualChannel
	for i, ch := range text {
		if ch < '0' || ch > '9' {
			return text[:i]
		}
	}
	return text
}
