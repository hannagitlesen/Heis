package distributor

import (
	"assigner"
	"config"
	le "localelevator"
	"peers"
	"time"
)

func InitDistributorElev(id string) config.DistributorElevator {
	requests := make([][]config.RequestsState, config.NumFloors)
	for floor := range requests {
		requests[floor] = make([]config.RequestsState, config.NumButtons)
	}

	return config.DistributorElevator{Requests: requests, ID: id, Floor: 0, Behaviour: config.Idle, Direction: config.MD_Stop}
}

func Broadcast(elevators map[string]*config.DistributorElevator, ch_NetworkMessageTx chan<- map[string]config.DistributorElevator) {
	elevatorsCopy := make(map[string]config.DistributorElevator, 0)
	for id, elev := range elevators {
		elevatorsCopy[id] = *elev
	}
	ch_NetworkMessageTx <- elevatorsCopy
	//Heihei må vi sleepe her?
}

func SetHallLights(elevators map[string]*config.DistributorElevator) {
	for _, elev := range elevators {
		for floor := range elev.Requests {
			for button := config.BT_HallUp; button <= config.BT_HallDown; button++ {
				lightsOn := false
				if elev.Requests[floor][button] == config.Confirmed {
					lightsOn = true
				}
				le.SetButtonLamp(le.ButtonType(button), floor, lightsOn)
			} 
		}
	}

}

func Distributor(
	myID string,
	ch_newLocalState chan le.Elevator,
	ch_newLocalOrder chan le.ButtonEvent,
	ch_orderToElev chan le.ButtonEvent,
	ch_arrivedAtFloors chan int,
	ch_obstr chan bool,
	ch_peerUpdate chan peers.PeerUpdate,
	ch_peerTxEnable chan bool,
	ch_NetworkMessageTx chan map[string]config.DistributorElevator,
	ch_NetworkMessageRx chan map[string]config.DistributorElevator,
	ch_watchdogPet chan bool,
	ch_watchdogBark chan bool) {

	elevators := make(map[string]*config.DistributorElevator)
	thisElevator := new(config.DistributorElevator)
	*thisElevator = InitDistributorElev(myID)
	elevators[myID] = thisElevator

	connectTimer := time.NewTimer(time.Duration(config.ConnectTimeout) * time.Second)

	select {
	case newElevators := <-ch_NetworkMessageRx:
		for ID, elev := range newElevators {
			if ID == myID {
				for floor := range elevators[myID].Requests {
					if elev.Requests[floor][config.BT_Cab] == config.Confirmed || elev.Requests[floor][config.BT_Cab] == config.Unconfirmed {
						ch_newLocalOrder <-le.ButtonEvent{floor, config.BT_Cab}
					}
					
				}
			} else { //Make sure that new elevator is updated on states
				elevators[ID] = &elev
			}
		}
	case <-connectTimer.C :
		break

	}
	for {
		select {
		case newState := <-ch_newLocalState:
			if newState.Behaviour == le.ElevBehaviour(config.Unavailable) {
				//kjøre reassign??
			}
			if newState.Floor != elevators[myID].Floor || newState.Behaviour == le.DoorOpen || newState.Behaviour == le.Idle { //Trenger vi å sjekke endring i direction?
				elevators[myID].Floor = newState.Floor
				ch_watchdogPet <-false
			}

			elevators[myID].Behaviour = config.ElevBehaviour(newState.Behaviour)
			elevators[myID].Direction = config.MotorDirection(newState.Direction)

			for floor := range newState.Requests {
				for button := range newState.Requests[floor] {
					if newState.Requests[floor][button] && elevators[myID].Requests[floor][button] == config.Unconfirmed && elevators[myID].Behaviour != config.Unavailable {
						elevators[myID].Requests[floor][button] = config.Confirmed
					}
					if !newState.Requests[floor][button] && elevators[myID].Requests[floor][button] == config.Confirmed {
						elevators[myID].Requests[floor][button] = config.Completed
					}
				}
			}
			Broadcast(elevators, ch_NetworkMessageTx)
			
		case newLocalOrder := <-ch_newLocalOrder:
			assigner.AssignOrder(elevators, newLocalOrder)
			if elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] == config.Unconfirmed {
				Broadcast(elevators, ch_NetworkMessageTx)
				elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] = config.Confirmed
				SetHallLights(elevators)
				ch_orderToElev <- newLocalOrder
			}

			Broadcast(elevators, ch_NetworkMessageTx)
			//SetHallLights(elevators)

		case updatedElevators := <-ch_NetworkMessageRx:
			//Updates the local map of elevators
			for ID, elev := range updatedElevators {
				for floor := range elev.Requests {
					for button := range elev.Requests[floor] {
						if !(updatedElevators[myID].Requests[floor][button] == config.Unconfirmed && elevators[myID].Requests[floor][button] == config.Confirmed) {
							elevators[ID].Requests[floor][button] = updatedElevators[ID].Requests[floor][button]	
						}
						if elevators[myID].Behaviour != config.Unavailable && elevators[myID].Requests[floor][button] == config.Unconfirmed  {
							elevators[myID].Requests[floor][button] = config.Confirmed
							SetHallLights(elevators)
							ch_orderToElev <- le.ButtonEvent{floor, le.ButtonType(button)}
							Broadcast(elevators, ch_NetworkMessageTx) //Skal vi flytt broadcast ut av for-loopen?
						}
						if elev.Requests[floor][button] == config.Completed {
							elevators[ID].Requests[floor][button] = config.None
							Broadcast(elevators, ch_NetworkMessageTx) //Skal vi flytt broadcast ut av for-loopen?
						}
					}
				}
				elevators[ID].Floor = updatedElevators[ID].Floor
				elevators[ID].Direction = updatedElevators[ID].Direction
				elevators[ID].Behaviour = updatedElevators[ID].Behaviour
			}
		// Check if there is an unknown elevator in the updated elevators that we should add to our map?

		
		case peerUpdate := <-ch_peerUpdate:

		case <-ch_watchdogBark:
			elevators[myID].Behaviour = config.Unavailable
			Broadcast(elevators, ch_NetworkMessageTx)
			for floor := range elevators[myID].Requests {
				for button := range elevators[myID].Requests[floor] {
					elevators[myID].Requests[floor][button] = config.None
				}
			}

		}

	}

}
