package pdnsdata

type ServerInfo struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	DaemonType string `json:"daemon_type"`
	Version    string `json:"version"`
	URL        string `json:"url"`
	ConfigURL  string `json:"config_url"`
	ZonesURL   string `json:"zones_url"`
}
