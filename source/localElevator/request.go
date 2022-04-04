package localelevator

import (
	"elevio"
)

func RequestsAbove(elev Elevator) bool {
	for f := elev.Floor + 1; f < len(elev.Requests); f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(elev Elevator) bool {
	for f := 0; f < elev.Floor; f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(elev Elevator) bool {
	for b := 0; b < 3; b++ {
		if elev.Requests[elev.Floor][b] {
			return true
		}
	}
	return false
}

func RequestsNextAction(elev *Elevator) {
	switch elev.Direction {
	case elevio.MD_Up:
		if RequestsAbove(*elev) {
			elev.Direction = elevio.MD_Up
			elev.Behaviour = Moving
		} else if RequestsHere(*elev) {
			elev.Direction = elevio.MD_Down
			elev.Behaviour = DoorOpen
		} else if RequestsBelow(*elev) {
			elev.Direction = elevio.MD_Down
			elev.Behaviour = Moving
		} else {
			elev.Direction = elevio.MD_Stop
			elev.Behaviour = Idle
		}
	case elevio.MD_Down:
		if RequestsBelow(*elev) {
			elev.Direction = elevio.MD_Down
			elev.Behaviour = Moving
		} else if RequestsHere(*elev) {
			elev.Direction = elevio.MD_Up
			elev.Behaviour = DoorOpen
		} else if RequestsAbove(*elev) {
			elev.Direction = elevio.MD_Up
			elev.Behaviour = Moving
		} else {
			elev.Direction = elevio.MD_Stop
			elev.Behaviour = Idle
		}
	case elevio.MD_Stop:
		if RequestsHere(*elev) {
			elev.Direction = elevio.MD_Stop
			elev.Behaviour = DoorOpen
		} else if RequestsAbove(*elev) {
			elev.Direction = elevio.MD_Up
			elev.Behaviour = Moving
		} else if RequestsBelow(*elev) {
			elev.Direction = elevio.MD_Down
			elev.Behaviour = Moving
		} else {
			elev.Direction = elevio.MD_Stop
			elev.Behaviour = Idle
		}
	}
}

func RequestsShouldStop(elev Elevator) bool {
	switch elev.Direction {
	case elevio.MD_Down:
		return elev.Requests[elev.Floor][elevio.BT_HallDown] || elev.Requests[elev.Floor][elevio.BT_Cab] || !RequestsBelow(elev)
	case elevio.MD_Up:
		return elev.Requests[elev.Floor][elevio.BT_HallUp] || elev.Requests[elev.Floor][elevio.BT_Cab] || !RequestsAbove(elev)
	default:
		return true
	}
}

func RequestsClearAtCurrentFloor(elev Elevator) {
	elev.Requests[elev.Floor][elevio.BT_Cab] = false
	switch elev.Direction {
	case elevio.MD_Up:
		if !RequestsAbove(elev) && !elev.Requests[elev.Floor][elevio.BT_HallUp] {
			elev.Requests[elev.Floor][elevio.BT_HallDown] = false
		}
		elev.Requests[elev.Floor][elevio.BT_HallUp] = false
	case elevio.MD_Down:
		if !RequestsBelow(elev) && !elev.Requests[elev.Floor][elevio.BT_HallDown] {
			elev.Requests[elev.Floor][elevio.BT_HallUp] = false
		}
		elev.Requests[elev.Floor][elevio.BT_HallDown] = false
	}
}
