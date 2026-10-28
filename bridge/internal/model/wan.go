package model

// Wan represents a gateway WAN interface with link, protocol, and traffic data.
type Wan struct {
	//Labels
	Port          int8   `json:"port"`
	Name          string `json:"name"`
	Desc          string `json:"portDesc"`
	Type          int8   `json:"type"`
	Ip            string `json:"ip"`
	Proto         string `json:"proto"`
	Status        int8   `json:"status"`
	InternetState int8   `json:"internetState"`
	LinkSpeed     int8   `json:"speed"`
	//Fields
	RxRate  float64 `json:"rxRate"`
	TxRate  float64 `json:"txRate"`
	Latency float64 `json:"latency"`
}

// GetLinkSpeed converts the encoded WAN link speed to Mbps and returns 0 when
// the interface is not active.
func (w *Wan) GetLinkSpeed() int32 {
	if 0 == w.Status {
		return 0
	}
	switch w.LinkSpeed {
	case 0:
		return 0
	case 1:
		return 10
	case 2:
		return 100
	case 3:
		return 1000
	case 4:
		return 2500
	case 5:
		return 10000
	case 6:
		return 5000
	case 7:
		return 25000
	case 8:
		return 100000
	case 9:
		return 40000
	default:
		return 0
	}
}

// GetStatus maps the WAN administrative status code to a readable label.
func (wan *Wan) GetStatus() string {
	switch wan.Status {
	case 0:
		return "Enabled"
	case 1:
		return "Disabled"
	default:
		return "Unknown"
	}
}

// GetType maps the WAN interface type code to its configured role label.
func (wan *Wan) GetType() string {
	switch wan.Type {
	case 0:
		return "WAN"
	case 1:
		return "WAN/LAN"
	case 2:
		return "LAN"
	}
	return "WAN"
}

// VirtualWan is an Omada 6.x virtual WAN: a VLAN sub-interface configured on a
// physical WAN port. The controller only reports it on the Web API gateway
// endpoint (virtualWanStats); the Open API wan-status and ISP load endpoints
// list physical ports only.
type VirtualWan struct {
	ID              string `json:"virtualWanId"`
	Name            string `json:"name"`
	Status          int8   `json:"status"`
	InternetState   int8   `json:"internetState"`
	OnlineDetection int8   `json:"onlineDetection"`
	Type            int8   `json:"type"`
	Proto           string `json:"proto"`
	Mac             string `json:"mac"`
	Ipv4            struct {
		Ip      string `json:"ip"`
		Gateway string `json:"gateway"`
	} `json:"wanPortIpv4Config"`
	// ParentPort is the physical WAN port carrying this virtual WAN (0 = unknown).
	ParentPort int8 `json:"-"`
}

// VirtualWanPortBase is the synthetic port number of the first virtual WAN.
// Physical gateway ports are 1..n (n <= 12 today), so 100+ never collides.
const VirtualWanPortBase int8 = 100

// AsWan renders the virtual WAN as a WAN row. Omada exposes no per-virtual-WAN
// rates or latency, so those stay 0; link speed is inherited from the parent port.
func (vw *VirtualWan) AsWan(port int8, linkSpeed int8) Wan {
	return Wan{
		Port:          port,
		Name:          vw.Name,
		Desc:          vw.Name,
		Type:          0,
		Ip:            vw.Ipv4.Ip,
		Proto:         vw.Proto,
		Status:        vw.Status,
		InternetState: vw.InternetState,
		LinkSpeed:     linkSpeed,
	}
}

// AsIsp renders the virtual WAN as an ISP row for the ISP collector.
func (vw *VirtualWan) AsIsp(port int8, gatewayName, gatewayMac string, gatewayStatus int8, maxBandwidth int32) Isp {
	status := int8(0)
	if vw.InternetState == 1 {
		status = 1
	}
	return Isp{
		GatewayName:   gatewayName,
		GatewayMac:    gatewayMac,
		GatewayStatus: gatewayStatus,
		Name:          vw.Name,
		Port:          port,
		Status:        status,
		IP:            vw.Ipv4.Ip,
		LoadBalance:   "0",
		MaxBandwidth:  maxBandwidth,
	}
}
