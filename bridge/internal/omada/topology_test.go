package omada

import (
	"encoding/json"
	"testing"
)

// Trimmed copy of a real OC220 6.x topology payload: gateway → switch (LAN2 ↔
// port 1) → switch over LAG (ports 5,6 both ends) → AP on port 1; controller
// hangs off the second switch and has no MAC-keyed inventory entry we care about.
const topologySample = `{"errorCode":0,"result":{"topologyArray":[
 {"type":"gateway","name":"ER8411","mac":"5C-62-8B-31-44-FF","successors":[
  {"type":"switch","name":"SX3008F","mac":"60-83-E7-82-17-26",
   "wiredUpInfo":{"port":{"port":"1"},"upLinkPort":{"port":"2","name":"LAN2"},"linkSpeed":5,"duplex":2},
   "successors":[
    {"type":"switch","name":"SX3206HPP","mac":"AC-15-A2-2D-83-0A",
     "upperPort":{"port":{"port":7,"lagId":1,"lagPorts":[5,6]},"upLinkPort":{"port":9,"lagId":1,"lagPorts":[5,6]},"linkSpeed":5},
     "wiredUpInfo":{"port":{"port":"7","lagId":"1","lagPorts":[5,6]},"upLinkPort":{"port":"9","lagId":"1","lagPorts":[5,6]},"linkSpeed":5},
     "successors":[
      {"type":"ap","name":"AP-1","mac":"10-5A-95-F6-4F-A9","successors":[],
       "wireUpLink":{"upLinkPort":{"port":1},"linkSpeed":4,"duplex":2}}
     ]},
    {"type":"switch","name":"SG3218XP","mac":"7C-F1-7E-17-0D-3B",
     "wiredUpInfo":{"port":{"port":"20","lagId":"2","lagPorts":[17,18]},"upLinkPort":{"port":"10","lagId":"2","lagPorts":[7,8]},"linkSpeed":5},
     "successors":[{"type":"omada controller","name":"Omada Controller","mac":"8C-86-DD-35-72-79","switchUpLink":{"upLinkPort":{"port":1},"linkSpeed":3}}]}
   ]}
 ]}]}}`

func TestParseTopology(t *testing.T) {
	var payload topologyResponse
	if err := json.Unmarshal([]byte(topologySample), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := parseTopology(payload)

	sw := got["60-83-E7-82-17-26"]
	if sw == nil || sw.ParentName != "ER8411" || sw.ParentPort != "2" || sw.LocalPort != "1" || sw.LinkSpeedMbps != 10000 {
		t.Fatalf("SX3008F uplink = %+v", sw)
	}
	lag := got["AC-15-A2-2D-83-0A"]
	if lag == nil || lag.ParentName != "SX3008F" || lag.ParentPort != "LAG 1" || lag.LagID != "1" || lag.LagPorts != "5,6" {
		t.Fatalf("SX3206HPP uplink = %+v", lag)
	}
	ap := got["10-5A-95-F6-4F-A9"]
	if ap == nil || ap.ParentName != "SX3206HPP" || ap.ParentPort != "1" || ap.LinkSpeedMbps != 2500 || ap.LocalPort != "" {
		t.Fatalf("AP-1 uplink = %+v", ap)
	}
	sg := got["7C-F1-7E-17-0D-3B"]
	if sg == nil || sg.ParentPort != "LAG 2" || sg.LagPorts != "17,18" {
		t.Fatalf("SG3218XP uplink = %+v", sg)
	}
	if ctl := got["8C-86-DD-35-72-79"]; ctl == nil || ctl.ParentName != "SG3218XP" || ctl.ParentPort != "1" {
		t.Fatalf("controller uplink = %+v", ctl)
	}
	if _, ok := got["5C-62-8B-31-44-FF"]; ok {
		t.Fatalf("gateway must not have an uplink")
	}
}
