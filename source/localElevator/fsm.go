package localelevator

import (
	"config"
	"elevio"
	"time"
)

func SetAllLocalLights(elev *Elevator) {
	elevio.SetFloorIndicator(elev.Floor)

	for floor := range elev.Requests {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, elev.Requests[floor][elevio.BT_Cab])
	}
}

func FSM(
	ch_newLocalState chan<- Elevator,
	ch_orderToFSM chan elevio.ButtonEvent,
	ch_resetLocalHallOrders chan bool,
	ch_arrivedAtFloors chan int,
	ch_obstr chan bool,
) {

	doorTimer := time.NewTimer(time.Duration(config.DoorTimerDuration) * time.Second)

	elev := NewElevator()
	e := &elev

	elevio.SetMotorDirection(elevio.MD_Down)
	for { 
		floor := <-ch_arrivedAtFloors
		elevio.SetMotorDirection(elevio.MD_Stop)
		e.Floor = floor
		break
	}
	SetAllLocalLights(e)
	elevio.SetDoorOpenLamp(false)

	timerUpdateStates := time.NewTimer(time.Duration(config.LocalStateUpdate) * time.Second)

	for {
		ch_newLocalState <- elev
		select {
		case order := <-ch_orderToFSM:
			switch elev.Behaviour {
			case DoorOpen:
				if elev.Floor == order.Floor {
					doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
				} else {
					elev.Requests[order.Floor][order.Button] = true
				}
			case Moving:
				elev.Requests[order.Floor][order.Button] = true
			case Idle:
				if elev.Floor == order.Floor {
					SetAllLocalLights(e)
					doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
					elevio.SetDoorOpenLamp(true)
					elev.Behaviour = DoorOpen
				} else {
					elev.Requests[order.Floor][order.Button] = true
					RequestsNextAction(e)
					elevio.SetMotorDirection(elev.Direction)
				}
			}
			SetAllLocalLights(e)

		case floor := <-ch_arrivedAtFloors:
			elev.Floor = floor
			switch elev.Behaviour {
			case Moving:
				if RequestsShouldStop(*e) {
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					elev.Behaviour = DoorOpen
					doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
					RequestsClearAtCurrentFloor(*e)
				}
			default:
				break
			}
			SetAllLocalLights(e)

		case <-doorTimer.C:
			if elev.Obstructed {
				doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
				break
			}

			switch elev.Behaviour {
			case DoorOpen:
				RequestsNextAction(e)

				switch elev.Behaviour {
				case DoorOpen:
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
					RequestsClearAtCurrentFloor(*e)
					SetAllLocalLights(e)
				case Moving:
					fallthrough
				case Idle:
					elevio.SetMotorDirection(elev.Direction)
					elevio.SetDoorOpenLamp(false)
					if elev.Direction == elevio.MD_Stop {
						elev.Behaviour = Idle
					} else {
						elev.Behaviour = Moving
					}
				}
			}

		case obstr := <-ch_obstr:
			elev.Obstructed = obstr
			if obstr && e.Behaviour == DoorOpen {
				doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
			}

		case <-timerUpdateStates.C:
			timerUpdateStates.Reset(time.Duration(config.LocalStateUpdate) * time.Second)

		case <-ch_resetLocalHallOrders:
			for floor := range elev.Requests {
				for button := config.BT_HallUp; button <= config.BT_HallDown; button++ {
					elev.Requests[floor][button] = false
				}
			}
		}
	}
}
