package main

// ToDo: Import an callout interfaces
// import (
//	"isc.org/stork/hooks/server/[PROVIDE-ME]callout"
// )

// Callouts structure
type callouts struct{}

// Closer interface implementation
func (c *callouts) Close() error {
	// ToDo: Implement close method. You should clean all used resources here.
	return nil
}

// Interface checks
// ToDo: Add an interface check below for all imported callouts.
// var _ [PROVIDE-ME]callout.[PROVIDE-ME]Callout = (*callouts)(nil)

// Interface implementations
// ToDo: Implement all imported interfaces.
