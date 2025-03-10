package testutil

import (
	"maps"
	"math/rand/v2"
	"slices"
)

// Zone information returned by random zones generator.
type Zone struct {
	Name   string
	Class  string
	Type   string
	Serial int64
}

// This function generates a collection of zones used in the benchmarks.
// The function argument specifies the number of zones to be generated.
//
//nolint:gosec
func generateRandomZones(existingZones []*Zone, num int, class, zoneType string, serial int64) []*Zone {
	// We construct labels from this set of characters.
	const charset = "abcdefghijklmnopqrstuvwxyz"
	// We want to simulate a use case where zones have child zones, e.g.:
	// example.com, zone1.example.com, etc. This slice will be used to
	// remember some of the previously generated parent zones and we
	// will derive child zones from it.
	labels := []string{}
	// Make sure the generated zones are unique.
	zones := make(map[string]*Zone)
	for _, existingZone := range existingZones {
		zones[existingZone.Name] = existingZone
	}
	// Generate num number of zones.
	var i int64
	for len(zones) < num+len(existingZones) {
		// For every 10th zone we forget previously generated zones
		// and generate completely new one.
		if i%10 == 0 {
			labels = []string{}
		}
		// Generated zone name will be stored here.
		var name string
		// Generate the name that has up to 6 labels.
		labelsCount := rand.IntN(5) + 1
		// Generate the labels.
		for j := 0; j < labelsCount; j++ {
			// Next label.
			var label string
			// If we cached some labels let's use them to generate
			// a child zone. Don't use the cached label if this is
			// the front label. Reusing front label would cause
			// zone name duplicates.
			if j < labelsCount-1 && len(labels) > j {
				label = labels[j]
			} else {
				// Label is not cached for this position, so we have to generate one.
				// The label length is between 1 and 16.
				for k := 0; k < rand.IntN(15)+1; k++ {
					// Get a random set of characters.
					label += string(charset[rand.IntN(len(charset)-1)])
				}
				// Cache this label for generating child zones.
				labels = append(labels, label)
			}
			// If this is not the last label we should add a dot as a separator.
			if j > 0 {
				label += "."
			}
			// Prepend the label and the dot to the name.
			name = label + name
		}
		// Only add the zone if it is not in conflict with the existing zones.
		if _, exists := zones[name]; !exists {
			// Create the zone entry.
			zones[name] = &Zone{
				Name:   name,
				Class:  class,
				Type:   zoneType,
				Serial: serial,
			}
			i++
		}
	}
	return slices.Collect(maps.Values(zones))
}

// Generate random zones with default parameters.
func GenerateRandomZones(num int) []*Zone {
	return generateRandomZones([]*Zone{}, num, "IN", "primary", 20240304)
}

// Generate more zones with default parameters.
func GenerateMoreZones(existingZones []*Zone, num int) []*Zone {
	return generateRandomZones(existingZones, num, "IN", "primary", 20240304)
}

// Generate more zones with a specific class.
func GenerateMoreZonesWithClass(existingZones []*Zone, num int, class string) []*Zone {
	return generateRandomZones(existingZones, num, class, "primary", 20240304)
}

// Generate more zones with a specific type.
func GenerateMoreZonesWithType(existingZones []*Zone, num int, zoneType string) []*Zone {
	return generateRandomZones(existingZones, num, "IN", zoneType, 20240304)
}

// Generate more zones with a specific serial.
func GenerateMoreZonesWithSerial(existingZones []*Zone, num int, serial int64) []*Zone {
	return generateRandomZones(existingZones, num, "IN", "primary", serial)
}
