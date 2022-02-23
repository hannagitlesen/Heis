
package elevator

import "elevio"

type ElevBehaviour int const (
	Idle = 0
	DoorOpen = 1
	Moving = 2

)

type Elevator struct {
	Floor int
	Direction elevio.MotorDirection
	Requests [][]bool
	Behaviour ElevBehaviour

}


