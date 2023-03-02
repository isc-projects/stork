package keaconfig

// Represents a client class in Kea configuration.
// todo: it currently only contains class name because it is
// all we need for current use cases. It will have extra fields
// when we need them.
type ClientClass struct {
	Name string `json:"name"`
}
