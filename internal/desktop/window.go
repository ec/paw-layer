package desktop

type Window struct {
	Address    string
	Class      string
	Title      string
	X          int
	Y          int
	Width      int
	Height     int
	Workspace  int
	Floating   bool
	Fullscreen bool
	Focused    bool
}
