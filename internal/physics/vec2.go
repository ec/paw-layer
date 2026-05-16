package physics

type Vec2 struct {
	X float64
	Y float64
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec2) Scale(factor float64) Vec2 {
	return Vec2{X: v.X * factor, Y: v.Y * factor}
}
