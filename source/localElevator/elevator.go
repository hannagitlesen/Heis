package localelevator

const NumFloors = 4
const NumButtons = 3

type ElevBehaviour int

const (
	Idle     ElevBehaviour = 0
	DoorOpen ElevBehaviour = 1
	Moving   ElevBehaviour = 2
)

type Elevator struct {
	Floor     int
	Direction MotorDirection
	Requests  [][NumButtons]bool
	Behaviour ElevBehaviour
}

func NewElevator() Elevator {
	e := Elevator{}
	e.Requests = make([][NumButtons]bool, NumFloors)
	return e
}
