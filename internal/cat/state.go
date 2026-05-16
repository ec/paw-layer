package cat

type Direction string

const (
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type State string

const (
	StateIdle  State = "idle"
	StateWalk  State = "walk"
	StateSleep State = "sleep"
	StatePeek  State = "peek"
	StateSit   State = "sit"
	StateRun   State = "run"
)
