package codegen

type Indentation int

const (
	tabs Indentation = iota
	fourSpaces
)

// Returns indentation coefficient. It is the number of characters within
// a single indentation.
func getIndentationCoefficient(kind Indentation) int {
	switch kind {
	case tabs:
		return 1
	case fourSpaces:
		return 4
	default:
		return 1
	}
}
