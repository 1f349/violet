package target

type Route struct {
	Pre  bool
	Host string
	Port int
	Path string
	Abs  bool
}
