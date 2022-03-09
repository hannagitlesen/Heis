package localelevator

import (
	"time"
)

func SetAllLocalLights(elev *Elevator) {
	SetFloorIndicator(elev.Floor)

	for f := range elev.Requests {
		SetButtonLamp(BT_Cab, f, elev.Requests[f][BT_Cab])
	}
	//FLYTTER HALL TIL DISTRIBUTOR
}

func NewOrder(elev *Elevator, order ButtonEvent, doorTimer *time.Timer) {
	switch elev.Behaviour {
	case DoorOpen:
		if elev.Floor == order.Floor {
			doorTimer.Reset(time.Duration(DoorTimerDuration) * time.Second)
		} else {
			elev.Requests[order.Floor][order.Button] = true
		}
	case Moving:
		elev.Requests[order.Floor][order.Button] = true
	case Idle:
		if elev.Floor == order.Floor {
			SetAllLocalLights(elev)
			doorTimer.Reset(time.Duration(DoorTimerDuration) * time.Second)
			SetDoorOpenLamp(true)
			elev.Behaviour = DoorOpen
			//BROADCAST NEWLOCAL STATE

		} else {
			elev.Requests[order.Floor][order.Button] = true
			RequestsNextAction(elev)
			SetMotorDirection(elev.Direction)
			//BROADCAST NEWLOCAL STATE
		}
	}
	SetAllLocalLights(elev)
}

func ArrivedAtFloor(elev *Elevator, floor int, doorTimer *time.Timer) {
	elev.Floor = floor
	//TRENGER VI LYS HER
	switch elev.Behaviour {
	case Moving:
		if RequestsShouldStop(*elev) {
			//TRENGER VI LYS HER
			SetMotorDirection(MD_Stop)
			SetDoorOpenLamp(true)
			elev.Behaviour = DoorOpen
			doorTimer.Reset(time.Duration(DoorTimerDuration) * time.Second)
			//BROADCAST NEWLOCAL STATE
			RequestsClearAtCurrentFloor(elev)
		}
	default:
		break
	}
}

func DoorTimeout(elev *Elevator, doorTimer *time.Timer) {
	if !elev.Obstructed {
		switch elev.Behaviour {
		case DoorOpen:
			RequestsNextAction(elev)
			SetMotorDirection(elev.Direction)
			SetDoorOpenLamp(false)
			if elev.Direction == MD_Stop {
				elev.Behaviour = Idle
				//BROADCAST
			} else {
				elev.Behaviour = Moving
				//BROADCAST
			}
		}
	} else {
		doorTimer.Reset(time.Duration(DoorTimerDuration) * time.Second)
	}
}

func FSM(
	ch_newLocalState chan ElevBehaviour,
	ch_orderToElev chan ButtonEvent,
	ch_arrivedAtFloors chan int,
	ch_obstr chan bool) { //SKAL VI GIDDE STOP?

	doorTimer := time.NewTimer(time.Duration(DoorTimerDuration) * time.Second)
	//TIMER UPDATE STATE?

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

	//BROADCAST NEW ELEVATOR ON NETWORK

	for {
		SetAllLocalLights(e)
		select {
		case order := <-ch_orderToElev:
			NewOrder(e, order, doorTimer)

		case floor := <-ch_arrivedAtFloors:
			ArrivedAtFloor(e, floor, doorTimer)

		case <-doorTimer.C:
			DoorTimeout(e, doorTimer)

		case obstr := <-ch_obstr: //MOTSATT OBSTR? (hardware feil)
			elev.Obstructed = !obstr
			if !obstr && e.Behaviour == DoorOpen {
				doorTimer.Reset(time.Duration(DoorTimerDuration) * time.Second)
				//OPPSTART AKTIV??
			}
		}
	}
}
