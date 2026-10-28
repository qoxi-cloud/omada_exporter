package omada

import "github.com/RCooLeR/omada_exporter/internal/model"

// virtualWans returns the gateway's virtual WANs with their parent (trunk) port resolved.
func (ps *gatewayResponse) virtualWans() []model.VirtualWan {
	if ps == nil {
		return nil
	}
	vws := make([]model.VirtualWan, len(ps.Result.VirtualWans))
	copy(vws, ps.Result.VirtualWans)
	for i := range vws {
		vws[i].ParentPort = ps.parentPortFor(vws[i].Name)
	}
	return vws
}

// parentPortFor finds the physical port whose PVID list offers the named virtual WAN.
func (ps *gatewayResponse) parentPortFor(name string) int8 {
	for _, conf := range ps.Result.PortConfigs {
		for _, pv := range conf.AvailablePvidNames {
			if pv.PvId != 0 && pv.PvName == name {
				return conf.Port
			}
		}
	}
	return 0
}

// trunkPorts lists physical ports that only exist to carry virtual WANs.
func trunkPorts(vws []model.VirtualWan) map[int8]bool {
	trunks := map[int8]bool{}
	for _, vw := range vws {
		if vw.ParentPort != 0 {
			trunks[vw.ParentPort] = true
		}
	}
	return trunks
}

// linkSpeedOf returns the encoded link speed of a physical gateway port.
func linkSpeedOf(ports []model.GatewayPort, port int8) int8 {
	for _, p := range ports {
		if p.Port == port {
			return p.LinkSpeed
		}
	}
	return 0
}

// mergeVirtualWans hides trunk-only physical WAN rows (they have no internet of
// their own) and appends one WAN row per virtual WAN with a synthetic port
// number, so Home Assistant and the cards show the real uplinks. Physical WAN
// ports without virtual WANs are untouched and keep appearing automatically.
func mergeVirtualWans(wans []model.Wan, ports []model.GatewayPort, vws []model.VirtualWan, latencyMs map[string]float64) []model.Wan {
	if len(vws) == 0 {
		return wans
	}
	trunks := trunkPorts(vws)
	out := make([]model.Wan, 0, len(wans)+len(vws))
	for _, w := range wans {
		if !trunks[w.Port] {
			out = append(out, w)
		}
	}
	for i, vw := range vws {
		w := vw.AsWan(model.VirtualWanPortBase+int8(i), linkSpeedOf(ports, vw.ParentPort))
		w.Latency = latencyMs[vw.Name]
		out = append(out, w)
	}
	return out
}

// mergeVirtualIsps applies the same trunk-hiding and virtual-WAN expansion to
// the ISP load rows of one gateway.
func mergeVirtualIsps(isps []model.Isp, vws []model.VirtualWan, gatewayName, gatewayMac string, gatewayStatus int8) []model.Isp {
	if len(vws) == 0 {
		return isps
	}
	trunks := trunkPorts(vws)
	bandwidthByPort := map[int8]int32{}
	out := make([]model.Isp, 0, len(isps)+len(vws))
	for _, isp := range isps {
		bandwidthByPort[isp.Port] = isp.MaxBandwidth
		if !trunks[isp.Port] {
			out = append(out, isp)
		}
	}
	for i, vw := range vws {
		out = append(out, vw.AsIsp(model.VirtualWanPortBase+int8(i), gatewayName, gatewayMac, gatewayStatus, bandwidthByPort[vw.ParentPort]))
	}
	return out
}
