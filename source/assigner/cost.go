package assigner

import (
	"config"
	le "localelevator"
)

func RequestsAbove(elev config.DistributorElevator) bool {
	for f := elev.Floor + 1; f < len(elev.Requests); f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] == config.Confirmed {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(elev config.DistributorElevator) bool {
	for f := 0; f < elev.Floor; f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] == config.Confirmed {
				return true
			}
		}
	}
	return false
}

func RequestsHere(elev config.DistributorElevator) bool {
	for b := 0; b < 3; b++ {
		if elev.Requests[elev.Floor][b] == config.Confirmed {
			return true
		}
	}
	return false
}

func RequestsNextAction(elev *config.DistributorElevator) {
	switch elev.Direction {
	case config.MD_Up:
		if RequestsAbove(*elev) {
			elev.Direction = config.MD_Up
			elev.Behaviour = config.Moving
		} else if RequestsHere(*elev) {
			elev.Direction = config.MD_Down
			elev.Behaviour = config.DoorOpen
		} else if RequestsBelow(*elev) {
			elev.Direction = config.MD_Down
			elev.Behaviour = config.Moving
		} else {
			elev.Direction = config.MD_Stop
			elev.Behaviour = config.Idle
		}
	case config.MD_Down:
		if RequestsBelow(*elev) {
			elev.Direction = config.MD_Down
			elev.Behaviour = config.Moving
		} else if RequestsHere(*elev) {
			elev.Direction = config.MD_Up
			elev.Behaviour = config.DoorOpen
		} else if RequestsAbove(*elev) {
			elev.Direction = config.MD_Up
			elev.Behaviour = config.Moving
		} else {
			elev.Direction = config.MD_Stop
			elev.Behaviour = config.Idle
		}
	case config.MD_Stop:
		if RequestsHere(*elev) {
			elev.Direction = config.MD_Stop
			elev.Behaviour = config.DoorOpen
		} else if RequestsAbove(*elev) {
			elev.Direction = config.MD_Up
			elev.Behaviour = config.Moving
		} else if RequestsBelow(*elev) {
			elev.Direction = config.MD_Down
			elev.Behaviour = config.Moving
		} else {
			elev.Direction = config.MD_Stop
			elev.Behaviour = config.Idle
		}
	}
}

func RequestsShouldStop(elev config.DistributorElevator) bool {
	switch elev.Direction {
	case config.MD_Down:
		return (elev.Requests[elev.Floor][config.BT_HallDown] == config.Confirmed) || elev.Requests[elev.Floor][config.BT_Cab] == config.Confirmed || !RequestsBelow(elev)
	case config.MD_Up:
		return elev.Requests[elev.Floor][config.BT_HallUp] == config.Confirmed || elev.Requests[elev.Floor][config.BT_Cab] == config.Confirmed || !RequestsAbove(elev)
	default:
		return true
	}
}

func RequestsClearAtCurrentFloor(elev *config.DistributorElevator) {
	elev.Requests[elev.Floor][config.BT_Cab] = config.None
	switch elev.Direction {
	case config.MD_Up:
		if !RequestsAbove(*elev) && elev.Requests[elev.Floor][config.BT_HallUp] == config.None {
			elev.Requests[elev.Floor][config.BT_HallDown] = config.None //Tar med de som skal ned
		}
		elev.Requests[elev.Floor][config.BT_HallUp] = config.None
	case config.MD_Down:
		if !RequestsBelow(*elev) && elev.Requests[elev.Floor][config.BT_HallDown] == config.None {
			elev.Requests[elev.Floor][config.BT_HallUp] = config.None //Tar med de som skal opp
		}
		elev.Requests[elev.Floor][config.BT_HallDown] = config.None
		//VI MÅ KANSKJE LEGGE TIL NOE HER
	}
}

func TimeToIdle(e *config.DistributorElevator, request le.ButtonEvent) int {
	duration := 0

	elev := new(config.DistributorElevator)
	*elev = *e

	elev.Requests[request.Floor][request.Button] = config.Confirmed

	switch elev.Behaviour {
	case config.Idle:
		RequestsNextAction(elev)
		if elev.Direction == config.MD_Stop {
			return duration
		}
	case config.Moving:
		duration += config.TravelTime / 2
		elev.Floor += int(elev.Direction)
	case config.DoorOpen:
		duration -= config.DoorTimerDuration / 2
	}

	for {
		if RequestsShouldStop(*elev) {
			RequestsClearAtCurrentFloor(elev)
			duration += config.DoorTimerDuration
			RequestsNextAction(elev)
			if elev.Direction == config.MD_Stop {
				return duration
			}
		}
		elev.Floor += int(elev.Direction)
		duration += config.TravelTime
	}
}
