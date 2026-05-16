package cat

import (
	"math"

	"github.com/ec/paw-layer/internal/physics"
)

type Cat struct {
	ID        string
	Name      string
	Position  physics.Vec2
	Velocity  physics.Vec2
	Direction Direction
	State     State
	Animation string
	Speed     float64
	Scale     float64
}

func New(id, name string, speed, scale float64) Cat {
	return Cat{
		ID:        id,
		Name:      name,
		Position:  physics.Vec2{X: 0, Y: 120},
		Velocity:  physics.Vec2{X: speed, Y: 0},
		Direction: DirectionRight,
		State:     StateWalk,
		Animation: "walk",
		Speed:     speed,
		Scale:     scale,
	}
}

func (c *Cat) Update(dt float64, maxX float64) {
	c.State = StateWalk
	c.Animation = "walk"
	c.Position = c.Position.Add(c.Velocity.Scale(dt))

	if c.Position.X > maxX {
		c.Position.X = maxX
		c.Velocity.X = -c.Speed
		c.Direction = DirectionLeft
	}

	if c.Position.X < 0 {
		c.Position.X = 0
		c.Velocity.X = c.Speed
		c.Direction = DirectionRight
	}
}

func (c *Cat) MoveToward(dt float64, target physics.Vec2) bool {
	return c.MoveTowardWithSpeed(dt, target, c.Speed, true)
}

func (c *Cat) MoveTowardWithSpeed(dt float64, target physics.Vec2, speed float64, sitOnArrival bool) bool {
	delta := physics.Vec2{X: target.X - c.Position.X, Y: target.Y - c.Position.Y}
	distance := math.Hypot(delta.X, delta.Y)
	arrivalRadius := 3.0
	if sitOnArrival {
		arrivalRadius = 6.0
	}
	if distance <= arrivalRadius {
		c.Position = target
		c.Velocity = physics.Vec2{}
		if sitOnArrival {
			c.State = StateSit
			c.Animation = "sit"
		} else {
			c.State = StateRun
			c.Animation = "walk"
		}
		return true
	}

	if speed <= 0 {
		speed = c.Speed
	}
	// Smooth arrival: keep full speed while far away, then ease down near
	// target so the cat does not snap/jitter before sitting.
	slowRadius := 96.0
	if !sitOnArrival {
		slowRadius = 48.0
	}
	speedFactor := 1.0
	if distance < slowRadius {
		speedFactor = math.Max(0.25, distance/slowRadius)
	}
	step := speed * speedFactor * dt
	if step >= distance {
		c.Position = target
	} else {
		c.Position.X += delta.X / distance * step
		c.Position.Y += delta.Y / distance * step
	}
	if delta.X < 0 {
		c.Direction = DirectionLeft
	} else if delta.X > 0 {
		c.Direction = DirectionRight
	}
	if sitOnArrival {
		c.State = StateWalk
	} else {
		c.State = StateRun
	}
	c.Animation = "walk"
	return false
}

func (c *Cat) Peek(position physics.Vec2) {
	c.Position = position
	c.Velocity = physics.Vec2{}
	c.State = StatePeek
	c.Animation = "peek"
}

func (c *Cat) Sleep() {
	c.Velocity = physics.Vec2{}
	c.State = StateSleep
	c.Animation = "sleep"
}

func (c *Cat) Hide() {
	c.State = StateIdle
	c.Animation = "idle"
}
