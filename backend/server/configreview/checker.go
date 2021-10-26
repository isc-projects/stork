package configreview

type checker struct {
	name    string
	checkFn func(*ReviewContext) (*Report, error)
}
