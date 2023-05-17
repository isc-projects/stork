package main

// TODO: Import an callout specification (interface).
// import (
//	"isc.org/stork/hooks/server/[PROVIDE-ME]callouts"
// )

// Callout carrier structure.
type calloutCarrier struct{}

// Closer interface implementation.
func (c *calloutCarrier) Close() error {
	// TODO: Implement close method. You should clean all used resources here.
	return nil
}

// Interface checks.
// TODO: Add an interface check below for all imported callout specifications.
// var _ [PROVIDE-ME]callouts.[PROVIDE-ME]Callouts = (*calloutCarrier)(nil)

// Interface implementations.
// TODO: Implement all imported interfaces.
