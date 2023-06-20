package utils

// Compilable is an interface for structs with an asynchronous compile method.
type Compilable interface {
	Compile()
}

// MultiCompilable is a slice of multiple Compilable interfaces.
type MultiCompilable []Compilable

// Compile loops over the slice of Compilable interfaces and calls Compile on
// each one.
func (m MultiCompilable) Compile() {
	for _, i := range m {
		i.Compile()
	}
}
