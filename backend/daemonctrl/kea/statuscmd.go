package keactrl

// A structure holding the response from Kea's `get-status` command.
type Status struct {
	CSVLeaseFile *string `json:"csv-lease-file,omitempty"`
}
