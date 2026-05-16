package desktop

type Monitor struct {
	ID      int
	Name    string
	X       int
	Y       int
	Width   int
	Height  int
	Scale   float64
	Focused bool
}
