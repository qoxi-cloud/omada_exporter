package omada

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/RCooLeR/omada_exporter/internal/model"
	"github.com/RCooLeR/omada_exporter/internal/omada/session"
	"github.com/rs/zerolog/log"
)

// TopologyInfo is what the topology map knows about one device.
type TopologyInfo struct {
	Uplink       *model.TopologyUplink
	ClientHealth *model.ClientHealth
}

// GetTopology returns the wired uplink and client-health split of every device
// on the site, keyed by upper-case MAC, as drawn by the controller's topology
// map. The web API is the only place Omada exposes which port a switch or AP
// hangs off.
func (c *Client) GetTopology() (map[string]*TopologyInfo, error) {
	return c.session.FetchCached("web:topology", c.getTopologyFresh)
}

func (c *Client) getTopologyFresh() (map[string]*TopologyInfo, error) {
	omadaCID, siteID := c.ContextIDs()
	url := fmt.Sprintf("%s/%s/api/v2/sites/%s/topology", c.session.Config.Host, omadaCID, siteID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.session.MakeLoggedInRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := session.ReadResponseBody(resp, "topology")
	if err != nil {
		return nil, err
	}
	log.Info().Msg("Received data from topology endpoint")
	log.Debug().Bytes("data", body).Msg("Received data from topology endpoint")
	if err := session.ValidateAPIResponse(body, "topology"); err != nil {
		return nil, err
	}
	var payload topologyResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return parseTopology(payload), nil
}

// applyTopology attaches the resolved uplink and client health to each device.
func applyTopology(devices []model.DevicesInterface, infos map[string]*TopologyInfo) {
	if len(infos) == 0 {
		return
	}
	for _, d := range devices {
		dev, ok := d.(model.DeviceInterface)
		if !ok {
			continue
		}
		if info, found := infos[strings.ToUpper(dev.GetMac())]; found {
			if info.Uplink != nil {
				dev.SetTopology(info.Uplink)
			}
			if info.ClientHealth != nil {
				dev.SetClientHealth(info.ClientHealth)
			}
		}
	}
}

// parseTopology walks the successor tree and records, for every node with a
// MAC, its client-health split and (for children) the parent and both port ends.
func parseTopology(payload topologyResponse) map[string]*TopologyInfo {
	result := map[string]*TopologyInfo{}
	var walk func(node topologyNode, parent *topologyNode)
	walk = func(node topologyNode, parent *topologyNode) {
		if node.Mac != "" {
			info := &TopologyInfo{ClientHealth: node.ClientHealth}
			if parent != nil {
				info.Uplink = node.uplink(parent)
			}
			if info.Uplink != nil || info.ClientHealth != nil {
				result[strings.ToUpper(node.Mac)] = info
			}
		}
		for _, child := range node.Successors {
			walk(child, &node)
		}
	}
	for _, root := range payload.Result.TopologyArray {
		walk(root, nil)
	}
	return result
}

type topologyResponse struct {
	Result struct {
		TopologyArray []topologyNode `json:"topologyArray"`
	} `json:"result"`
}

type topologyNode struct {
	Type         string              `json:"type"`
	Name         string              `json:"name"`
	Mac          string              `json:"mac"`
	Successors   []topologyNode      `json:"successors"`
	ClientHealth *model.ClientHealth `json:"clientHealth"`
	// switch / gateway children
	WiredUpInfo *topologyLink `json:"wiredUpInfo"`
	UpperPort   *topologyLink `json:"upperPort"`
	// access point children
	WireUpLink *topologyLink `json:"wireUpLink"`
	// controller hanging off a switch
	SwitchUpLink *topologyLink `json:"switchUpLink"`
}

// topologyLink describes one wired hop: `port` is the child's own port (or
// LAG), `upLinkPort` the port on the parent. Omada mixes strings and numbers.
type topologyLink struct {
	Port       *topologyPort `json:"port"`
	UpLinkPort *topologyPort `json:"upLinkPort"`
	LinkSpeed  int8          `json:"linkSpeed"`
}

type topologyPort struct {
	Port     flexString   `json:"port"`
	Name     string       `json:"name"`
	LagID    flexString   `json:"lagId"`
	LagPorts []flexString `json:"lagPorts"`
}

func (n topologyNode) uplink(parent *topologyNode) *model.TopologyUplink {
	link := n.WiredUpInfo
	if link == nil {
		link = n.UpperPort
	}
	if link == nil {
		link = n.WireUpLink
	}
	if link == nil {
		link = n.SwitchUpLink
	}
	if link == nil {
		return nil
	}
	up := &model.TopologyUplink{
		ParentMac:     strings.ToUpper(parent.Mac),
		ParentName:    parent.Name,
		ParentType:    parent.Type,
		LinkSpeedMbps: (&model.Uplink{LinkSpeed: link.LinkSpeed}).GetLinkSpeed(),
	}
	if link.UpLinkPort != nil {
		up.ParentPort = string(link.UpLinkPort.Port)
		if link.UpLinkPort.LagID != "" {
			up.ParentPort = "LAG " + string(link.UpLinkPort.LagID)
		}
	}
	if link.Port != nil {
		up.LocalPort = string(link.Port.Port)
		up.LagID = string(link.Port.LagID)
		members := make([]string, 0, len(link.Port.LagPorts))
		for _, m := range link.Port.LagPorts {
			members = append(members, string(m))
		}
		up.LagPorts = strings.Join(members, ",")
	}
	return up
}

// flexString accepts JSON numbers or strings and keeps the textual form.
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	if i, err := n.Int64(); err == nil {
		*f = flexString(strconv.FormatInt(i, 10))
		return nil
	}
	*f = flexString(n.String())
	return nil
}
