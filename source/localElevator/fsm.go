package localelevator

import (
	"config"
	"fmt"
	"time"
)

func SetAllLocalLights(elev *Elevator) {
	SetFloorIndicator(elev.Floor)

	for f := range elev.Requests {
		SetButtonLamp(BT_Cab, f, elev.Requests[f][BT_Cab])
	}
}

func FSM(
	ch_newLocalState chan<- Elevator,
	ch_orderToElev chan ButtonEvent,
	ch_arrivedAtFloors chan int,
	ch_obstr chan bool) { //SKAL VI GIDDE STOP?

	doorTimer := time.NewTimer(time.Duration(config.DoorTimerDuration) * time.Second)

	elev := NewElevator()
	e := &elev

	SetMotorDirection(MD_Down)
	for { //Sender heis ned til nærmeste etasje
		floor := <-ch_arrivedAtFloors
		SetMotorDirection(MD_Stop)
		e.Floor = floor
		break
	}
	SetAllLocalLights(e)
	SetDoorOpenLamp(false)

	timerUpdateStates := time.NewTimer(time.Duration(config.UpdateTimeout) * time.Second)

	for {
		select {
		case order := <-ch_orderToElev:
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
					//elev.Requests[order.Floor][order.Button] = true
					SetAllLocalLights(e)
					doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
					SetDoorOpenLamp(true)
					elev.Behaviour = DoorOpen
				} else {
					elev.Requests[order.Floor][order.Button] = true
					RequestsNextAction(e)
					SetMotorDirection(elev.Direction)
				}
			}
			ch_newLocalState <- elev
			SetAllLocalLights(e)

		case floor := <-ch_arrivedAtFloors:
			elev.Floor = floor
			switch elev.Behaviour {
			case Moving:
				if RequestsShouldStop(*e) {
					SetMotorDirection(MD_Stop)
					SetDoorOpenLamp(true)
					elev.Behaviour = DoorOpen
					doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
					RequestsClearAtCurrentFloor(e)
				}
			default:
				break
			}
			SetAllLocalLights(e)
			ch_newLocalState <- elev
			fmt.Println("Arrived new floor")

		case <-doorTimer.C:
			if !elev.Obstructed {
				switch elev.Behaviour {
				case DoorOpen:
					RequestsNextAction(e)
					SetMotorDirection(elev.Direction)
					SetDoorOpenLamp(false)
					if elev.Direction == MD_Stop {
						elev.Behaviour = Idle
					} else {
						elev.Behaviour = Moving
					}
				}
			} else {
				doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
			}
			ch_newLocalState <- elev

		case obstr := <-ch_obstr: //MOTSATT OBSTR? (hardware feil)
			elev.Obstructed = obstr
			if obstr && e.Behaviour == DoorOpen {
				doorTimer.Reset(time.Duration(config.DoorTimerDuration) * time.Second)
				//OPPSTART AKTIV??
			}

		case <-timerUpdateStates.C:
			ch_newLocalState <- elev
			timerUpdateStates.Reset(time.Duration(config.UpdateTimeout) * time.Second)
		}
	}
}
