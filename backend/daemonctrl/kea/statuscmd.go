package keactrl

import (
	"encoding/json"
)

// A structure holding the response from Kea's `get-status` command.
type Status struct {
	CSVLeaseFile *string `json:"csv-lease-file,omitempty"`
}

// Construct a new Status from the raw bytes returned by Kea.
func NewStatus(raw []byte) (*Status, error) {
	var status Status
	err := json.Unmarshal(raw, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}
