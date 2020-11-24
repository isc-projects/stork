package dbmodel

// Structure representing a container for subnets. The container allows
// for accessing stored subnets using various keys (indexes). This
// significantly improves subnet lookup time comparing to the case when
// subnets are stored as a slice. New indexes can be added as needed
// in the future.
type IndexedSubnets struct {
	RandomAccess []Subnet
	// Index to be used when accessing subnets by prefix.
	ByPrefix map[string]*Subnet
}

// Creates new instance of the IndexedSubnets structure. It takes a
// slice of subnets as input. This slice will be used to build indexes.
func NewIndexedSubnets(subnets []Subnet) *IndexedSubnets {
	return &IndexedSubnets{
		RandomAccess: subnets,
	}
}

// Rebuild indexes using subnets stored in RandomAccess field as input.
// It returns false if the duplicates are found.
func (is *IndexedSubnets) Populate() bool {
	byPrefix := make(map[string]*Subnet)
	for i := range is.RandomAccess {
		if _, ok := byPrefix[is.RandomAccess[i].Prefix]; ok {
			return false
		}
		byPrefix[is.RandomAccess[i].Prefix] = &is.RandomAccess[i]
	}
	is.ByPrefix = byPrefix

	return true
}
