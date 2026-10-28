package model

// Switch represents a managed switch with uplink, port, LAG, and PoE telemetry.
type Switch struct {
	Device
	Uplink Uplink `json:"uplink"`
	Ports  []Port `json:"ports"`
	Lags   []Lag  `json:"lags"`
	//Labels
	FanStatus  int8  `json:"fanStatus"`
	PoeSupport bool  `json:"poeSupport"`
	PortNumber int8  `json:"portNum"`
	TotalPower int32 `json:"totalPower"`
	// Omada 6.x lists report the PoE budget as poeTotalPower
	PoeTotalPower float64 `json:"poeTotalPower"`
	LoopbackNum   int     `json:"loopbackNum"`
	BlockNum      int     `json:"blockNum"`
	//Fields
	PoeRemain float64 `json:"poeRemain"`
}

// GetTotalPowerWatts returns the PoE budget, preferring the 6.x poeTotalPower field.
func (sw *Switch) GetTotalPowerWatts() float64 {
	if sw.PoeTotalPower > 0 {
		return sw.PoeTotalPower
	}
	return float64(sw.TotalPower)
}

// GetPoeSupport converts the PoE support flag to a Yes or No label.
func (sw *Switch) GetPoeSupport() string {
	switch sw.PoeSupport {
	case true:
		return "Yes"
	case false:
		return "No"
	}
	return ""
}
