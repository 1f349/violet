package utils

type Compilable interface {
	Compile()
}

type MultiCompilable []Compilable

func (m MultiCompilable) Compile() {
	for _, i := range m {
		i.Compile()
	}
}
