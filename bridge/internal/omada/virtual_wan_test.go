package omada

import (
	"encoding/json"
	"testing"

	"github.com/RCooLeR/omada_exporter/internal/model"
)

const gatewayDetailFixture = `{"result":{
  "portStats":[{"port":1,"name":"SFP+ WAN1","status":1,"speed":5},{"port":4,"name":"WAN/LAN4","status":0,"speed":1}],
  "portConfigs":[
    {"port":1,"portCap":[{"linkSpeed":5}],"availablePvidNames":[{"pvId":0,"pvName":"Default"},{"pvId":902,"pvName":"LinkCom_sfp2"},{"pvId":903,"pvName":"Kopiyka_sfp3"}]},
    {"port":4,"portCap":[{"linkSpeed":3}],"availablePvidNames":[{"pvId":1,"pvName":"System"}]}
  ],
  "virtualWanStats":[
    {"virtualWanId":"a","name":"LinkCom_sfp2","status":1,"internetState":1,"proto":"dhcp","wanPortIpv4Config":{"ip":"195.72.145.224","gateway":"195.72.146.5"}},
    {"virtualWanId":"b","name":"Kopiyka_sfp3","status":1,"internetState":0,"proto":"dhcp","wanPortIpv4Config":{"ip":"0.0.0.0","gateway":"0.0.0.0"}}
  ]}}`

func fixtureDetail(t *testing.T) *gatewayResponse {
	t.Helper()
	d := &gatewayResponse{}
	if err := json.Unmarshal([]byte(gatewayDetailFixture), d); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return d
}

func TestVirtualWansResolveParentPort(t *testing.T) {
	vws := fixtureDetail(t).virtualWans()
	if len(vws) != 2 {
		t.Fatalf("want 2 virtual wans, got %d", len(vws))
	}
	for _, vw := range vws {
		if vw.ParentPort != 1 {
			t.Errorf("%s: parent port = %d, want 1", vw.Name, vw.ParentPort)
		}
	}
}

func TestMergeVirtualWansHidesTrunkAndKeepsOtherPorts(t *testing.T) {
	d := fixtureDetail(t)
	wans := []model.Wan{{Port: 1, Name: "SFP+ WAN1", Status: 1, InternetState: 1}, {Port: 4, Name: "WAN/LAN4", Status: 1}}
	merged := mergeVirtualWans(wans, d.Result.Ports, d.virtualWans(), map[string]float64{"LinkCom_sfp2": 1.5})
	names := map[string]model.Wan{}
	for _, w := range merged {
		names[w.Name] = w
	}
	if _, trunk := names["SFP+ WAN1"]; trunk {
		t.Errorf("trunk port must be hidden: %+v", merged)
	}
	if _, ok := names["WAN/LAN4"]; !ok {
		t.Errorf("physical WAN without virtual WANs must stay: %+v", merged)
	}
	lc, ok := names["LinkCom_sfp2"]
	if !ok || lc.Port != model.VirtualWanPortBase || lc.Ip != "195.72.145.224" || lc.InternetState != 1 || lc.LinkSpeed != 5 || lc.Latency != 1.5 {
		t.Errorf("LinkCom row wrong: %+v", lc)
	}
	if kp := names["Kopiyka_sfp3"]; kp.Port != model.VirtualWanPortBase+1 || kp.InternetState != 0 {
		t.Errorf("Kopiyka row wrong: %+v", kp)
	}
}

func TestMergeVirtualWansNoVirtualWansIsNoop(t *testing.T) {
	wans := []model.Wan{{Port: 1, Name: "WAN1"}}
	if got := mergeVirtualWans(wans, nil, nil, nil); len(got) != 1 || got[0].Name != "WAN1" {
		t.Errorf("unexpected change: %+v", got)
	}
}

func TestMergeVirtualIsps(t *testing.T) {
	d := fixtureDetail(t)
	isps := []model.Isp{{Port: 1, Name: "SFP+ WAN1", MaxBandwidth: 10000, Status: 2}}
	merged := mergeVirtualIsps(isps, d.virtualWans(), "ER8411", "5C-62-8B-31-44-FF", 1)
	if len(merged) != 2 {
		t.Fatalf("want 2 isp rows, got %d: %+v", len(merged), merged)
	}
	for _, isp := range merged {
		if isp.Port == 1 {
			t.Errorf("trunk ISP row must be hidden")
		}
		if isp.GatewayMac != "5C-62-8B-31-44-FF" || isp.MaxBandwidth != 10000 {
			t.Errorf("gateway context/bandwidth not inherited: %+v", isp)
		}
	}
	if merged[0].Name != "LinkCom_sfp2" || merged[0].Status != 1 || merged[1].Status != 0 {
		t.Errorf("status mapping wrong: %+v", merged)
	}
}
