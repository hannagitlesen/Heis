package localelevator

func SetAllLocalLights(elev *Elevator) {
	SetFloorIndicator(elev.Floor)

	for f := range elev.Requests {
		SetButtonLamp(BT_Cab, elev.Floor, elev.Requests[f][BT_Cab])
	}
	//FLYTTER HALL TIL DISTRIBUTOR
}

func NewOrder(elev *Elevator, order ButtonEvent) {
	switch elev.Behaviour {
	case DoorOpen:
		if elev.Floor == order.Floor {
			//START DOOR TIMER
		} else {
			elev.Requests[order.Floor][order.Button] = true
		}
	case Moving:
		elev.Requests[order.Floor][order.Button] = true
	case Idle:
		if elev.Floor == order.Floor {
			SetAllLocalLights(elev)
			//START TIMER
			SetDoorOpenLamp(true)
			elev.Behaviour = DoorOpen
			//BROADCAST NEWLOCALSTATE

		} else {
			elev.Requests[order.Floor][order.Button] = true
			RequestsNextAction(elev)
			SetMotorDirection(elev.Direction)
			//BROADCAST NEWLOCAL STATE
		}

	}
}

func ArrivedAtFloor(elev *Elevator, floor int) {
	elev.Floor = floor
	//TRENGER VI LYS HER
	switch elev.Behaviour {
	case Moving:
		if RequestsShouldStop(*elev) {
			//TRENGER VI LYS HER
			SetMotorDirection(MD_Stop)
			SetDoorOpenLamp(true)
			elev.Behaviour = DoorOpen
			//START TIMER
			//BROADCAST NEWLOCAL STATE
			RequestsClearAtCurrentFloor(elev)
		}
	default:
		break
	}
}

func DoorTimeout(elev *Elevator) {
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
}

