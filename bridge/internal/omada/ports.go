package omada

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RCooLeR/omada_exporter/internal/model"
	"github.com/RCooLeR/omada_exporter/internal/omada/session"
	"github.com/rs/zerolog/log"
)

// GetPortsAndLags fetches switch details from the Web API and copies the
// decoded port, LAG, uplink, traffic, power, and temperature fields onto sw.
func (c *Client) GetPortsAndLags(sw *model.Switch) error {
	omadaCID, siteID := c.ContextIDs()
	url := fmt.Sprintf("%s/%s/api/v2/sites/%s/switches/%s", c.session.Config.Host, omadaCID, siteID, sw.Mac)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.session.MakeLoggedInRequest(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := session.ReadResponseBody(resp, "switch ports")
	if err != nil {
		return err
	}
	log.Info().Msg(fmt.Sprintf("Received data from ports endpoint for %s", sw.Mac))
	log.Debug().Bytes("data", body).Msg("Received data from ports endpoint")
	if err := session.ValidateAPIResponse(body, "switch ports"); err != nil {
		return err
	}

	portdata := portResponse{}
	if err := json.Unmarshal(body, &portdata); err != nil {
		return err
	}
	sw.Temp = portdata.Result.Temp
	sw.TotalPower = portdata.Result.TotalPower
	sw.Uplink = portdata.Result.Uplink
	sw.RxRate = portdata.Result.RxRate
	sw.TxRate = portdata.Result.TxRate
	sw.Ports = portdata.Result.Ports
	sw.Lags = portdata.Result.Lags
	return nil
}

// GetApPorts fetches the LAN port list for the provided access point and stores
// the decoded ports on ap.
func (c *Client) GetApPorts(ap *model.AccessPoint) error {
	omadaCID, siteID := c.ContextIDs()
	url := fmt.Sprintf("%s/%s/api/v2/sites/%s/eaps/%s/ports", c.session.Config.Host, omadaCID, siteID, ap.Mac)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.session.MakeLoggedInRequest(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := session.ReadResponseBody(resp, "access point ports")
	if err != nil {
		return err
	}
	log.Info().Msg(fmt.Sprintf("Received data from ports endpoint for AP %s", ap.Mac))
	log.Debug().Bytes("data", body).Msg("Received data from ports endpoint for AP")
	if err := session.ValidateAPIResponse(body, "access point ports"); err != nil {
		return err
	}

	portdata := apPortResponse{}
	if err := json.Unmarshal(body, &portdata); err != nil {
		return err
	}
	ap.Ports = portdata.Result
	return nil
}

// GetGatewayPorts fetches gateway port telemetry, derives each port's maximum
// supported speed from the capability list, and stores the result on gw.
func (c *Client) GetGatewayPorts(gw *model.Gateway) error {
	portdata, err := c.getGatewayDetail(gw.Mac)
	if err != nil {
		return err
	}
	ports := make([]model.GatewayPort, len(portdata.Result.Ports))
	copy(ports, portdata.Result.Ports)
	for i, p := range ports {
		ports[i].MaxSpeed = portdata.GetMaxLinkSpeed(p.Port)
	}
	gw.Temp = portdata.Result.Temp
	gw.Ports = ports
	gw.RxRate = portdata.Result.RxRate
	gw.TxRate = portdata.Result.TxRate
	gw.VirtualWans = portdata.virtualWans()
	gw.Wans = mergeVirtualWans(gw.Wans, gw.Ports, gw.VirtualWans)
	return nil
}

// getGatewayDetail fetches the Web API gateway detail (ports, virtual WANs,
// temperature) once per cache window; the ISP collector reuses it.
func (c *Client) getGatewayDetail(mac string) (*gatewayResponse, error) {
	return c.session.FetchCached("web:gateway:"+mac, func() (*gatewayResponse, error) {
		omadaCID, siteID := c.ContextIDs()
		url := fmt.Sprintf("%s/%s/api/v2/sites/%s/gateways/%s", c.session.Config.Host, omadaCID, siteID, mac)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.session.MakeLoggedInRequest(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		body, err := session.ReadResponseBody(resp, "gateway ports")
		if err != nil {
			return nil, err
		}
		log.Info().Msg(fmt.Sprintf("Received data from ports endpoint for %s", mac))
		log.Debug().Bytes("data", body).Msg("Received data from ports endpoint")
		if err := session.ValidateAPIResponse(body, "gateway ports"); err != nil {
			return nil, err
		}

		portdata := &gatewayResponse{}
		if err := json.Unmarshal(body, portdata); err != nil {
			return nil, err
		}
		return portdata, nil
	})
}

// portResponse wraps the Web API payload returned for switch details.
type portResponse struct {
	Result model.Switch `json:"result"`
}

// apPortResponse wraps the Web API payload returned for access point LAN ports.
type apPortResponse struct {
	Result []model.AccessPointPort `json:"result"`
}

// gatewayResponse wraps the Web API payload returned for gateway port details.
type gatewayResponse struct {
	Result struct {
		Temp        float64             `json:"temp"`
		RxRate      float64             `json:"rxRate"`
		TxRate      float64             `json:"txRate"`
		PortConfigs []gatewayPortConfig `json:"portConfigs"`
		Ports       []model.GatewayPort `json:"portStats"`
		VirtualWans []model.VirtualWan  `json:"virtualWanStats"`
	} `json:"result"`
}

// gatewayPortConfig stores per-port capability data used to derive max speed.
type gatewayPortConfig struct {
	Port               int8       `json:"port"`
	PortCaps           []PortCap  `json:"portCap"`
	AvailablePvidNames []pvidName `json:"availablePvidNames"`
}

// pvidName is one VLAN a WAN port can carry; pvId != 0 entries are virtual WANs.
type pvidName struct {
	PvId   int    `json:"pvId"`
	PvName string `json:"pvName"`
}

// PortCap represents one advertised speed capability for a gateway port.
type PortCap struct {
	LinkSpeed int8
}

// GetMaxLinkSpeed scans the configured capabilities for a port and returns the
// fastest supported link speed in Mbps.
func (ps *gatewayResponse) GetMaxLinkSpeed(port int8) int32 {
	maxSpeed := 0
	for _, conf := range ps.Result.PortConfigs {
		if conf.Port == port {
			for _, cap := range conf.PortCaps {
				capSpeed := 0
				switch cap.LinkSpeed {
				case 0:
					capSpeed = 0
				case 1:
					capSpeed = 10
				case 2:
					capSpeed = 100
				case 3:
					capSpeed = 1000
				case 4:
					capSpeed = 2500
				case 5:
					capSpeed = 10000
				case 6:
					capSpeed = 5000
				case 7:
					capSpeed = 25000
				case 8:
					capSpeed = 100000
				case 9:
					capSpeed = 40000
				}
				if capSpeed > maxSpeed {
					maxSpeed = capSpeed
				}
			}
		}
	}
	return int32(maxSpeed)
}
