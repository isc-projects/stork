package main

// ToDo: Import an callout interfaces
// import (
//	"isc.org/stork/hooks/server/[PROVIDE-ME]callout"
// )

// Callout structure.
type callout struct{}

// Closer interface implementation.
func (c *callout) Close() error {
	// ToDo: Implement close method. You should clean all used resources here.
	return nil
}

// Interface checks.
// ToDo: Add an interface check below for all imported callouts.
// var _ [PROVIDE-ME]callout.[PROVIDE-ME]Callout = (*callout)(nil)

// Interface implementations.
// ToDo: Implement all imported interfaces.
