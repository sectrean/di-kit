package di

type Lifetime uint8

const (
	Singleton Lifetime = iota
	Transient Lifetime = iota
	Scoped    Lifetime = iota
)
