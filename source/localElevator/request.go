package request

import "Heis/elevator"

func RequestsAbove(elev elevator.Elevator) bool {
	for f := elev.Floor + 1; f < elevio._numFloors; f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(elev elevator.Elevator) bool {
	for f := 0; f < elev.Floor; f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(elev elevator.Elevator) bool {
	for btn := range elev.Floor {
		if elev.Floor {
			return true
		}
	}

	return false

}

func RequestsNextAction(elev elevator.Elevator) {

}

func RequestsShouldStop(elev elevator.Elevator) {

}

func RequestsClearAtCurrentFloor(elev elevator.Elevator) {

}
