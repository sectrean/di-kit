package di

type Lifetime uint

const (
	Singleton Lifetime = iota
	Transient Lifetime = iota
	Scoped    Lifetime = iota
)
