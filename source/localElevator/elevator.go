package localelevator

import (
	"config"
	"elevio"
)

type ElevBehaviour int

const (
	Idle     ElevBehaviour = 0
	DoorOpen ElevBehaviour = 1
	Moving   ElevBehaviour = 2
)

type Elevator struct {
	Floor     int
	Direction elevio.MotorDirection
	Requests  [][config.NumButtons]bool
	Behaviour ElevBehaviour
	Obstructed bool
}

func NewElevator() Elevator {
	elev := Elevator{}
	elev.Requests = make([][config.NumButtons]bool, config.NumFloors)
	return elev
}
