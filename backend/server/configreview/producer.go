package configreview

type producer struct {
	name      string
	produceFn func(*reviewContext) (*report, error)
}
