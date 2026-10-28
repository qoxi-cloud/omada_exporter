package model

// TopologyUplink is the wired parent of a device as drawn by the controller's
// topology map: which device it hangs off, the port on that parent, the local
// port (or LAG) on this device and the negotiated speed.
type TopologyUplink struct {
	ParentMac     string
	ParentName    string
	ParentType    string
	ParentPort    string // port number on the parent (or LAG id when LagID != "")
	LocalPort     string // port number on this device
	LagID         string
	LagPorts      string // comma separated member ports on this device
	LinkSpeedMbps int32
}
