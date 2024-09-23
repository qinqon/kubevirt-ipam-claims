package migration

type L2MigrationArgs struct {
	PortName string `json:"portName,omitempty"`
	PodRole  string `json:"podRole,omitempty"`
	State    string `json:"state,omitempty"`
}
