package configreview

type producer struct {
	name      string
	produceFn func(*ReviewContext) (*Report, error)
}
