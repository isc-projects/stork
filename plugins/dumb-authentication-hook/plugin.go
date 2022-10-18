package main

type Callout interface {
	Close() error
}

func Load() (any, error) {
	return &callout{}, nil
}

func Version() (string, string) {
	return "Stork Server", "1.7.0"
}
