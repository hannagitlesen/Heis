package distributor

import (
	"assigner"
	"config"
	"fmt"
	le "localelevator"
	"peers"
	"time"
)

func InitDistributorElev() config.DistributorElevator {
	requests := make([][]config.RequestsState, config.NumFloors)
	for floor := range requests {
		requests[floor] = make([]config.RequestsState, config.NumButtons)
	}

	return config.DistributorElevator{Requests: requests, Floor: 0, Behaviour: config.Idle, Direction: config.MD_Stop}
}

func Broadcast(elevators map[string]*config.DistributorElevator, ch_NetworkMessageTx chan<- map[string]config.DistributorElevator) {
	elevatorsCopy := make(map[string]config.DistributorElevator, 0)
	for id, elev := range elevators {
		elevatorsCopy[id] = *elev
	}
	ch_NetworkMessageTx <- elevatorsCopy
	//Heihei må vi sleepe her?
	time.Sleep(time.Millisecond * 50)
}

func SetHallLights(elevators map[string]*config.DistributorElevator) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for button := config.BT_HallUp; button <= config.BT_HallDown; button++ {
			lightsOn := false
			for _, elev := range elevators {
				if elev.Requests[floor][button] == config.Confirmed {
					lightsOn = true
				}
			}
			le.SetButtonLamp(le.ButtonType(button), floor, lightsOn)
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
	*thisElevator = InitDistributorElev()
	elevators[myID] = thisElevator

	connectTimer := time.NewTimer(time.Duration(config.ConnectTimeout) * time.Second)

	select {
	case newElevators := <-ch_NetworkMessageRx:
		for ID, elev := range newElevators {
			if ID == myID {
				for floor := range elevators[myID].Requests {
					if elev.Requests[floor][config.BT_Cab] == config.Confirmed || elev.Requests[floor][config.BT_Cab] == config.Unconfirmed {
						ch_newLocalOrder <- le.ButtonEvent{floor, config.BT_Cab}
					}
				}
			} else { //Make sure that new elevator is updated on states
				elevators[ID] = &elev
				fmt.Println("New ID pre loop")
			}
		}
		Broadcast(elevators, ch_NetworkMessageTx)
		break
	case <-connectTimer.C:
		break
	}
	for {
		select {
		case newLocalOrder := <-ch_newLocalOrder:
			assigner.AssignOrder(elevators, newLocalOrder)
			if elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] == config.Unconfirmed {
				Broadcast(elevators, ch_NetworkMessageTx)
				elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] = config.Confirmed
				SetHallLights(elevators)
				ch_orderToElev <- newLocalOrder
			}

			Broadcast(elevators, ch_NetworkMessageTx)

		case newState := <-ch_newLocalState:
			fmt.Println("New local state")
			fmt.Println(newState)
			fmt.Println(elevators[myID])
			//if newState.Behaviour == le.ElevBehaviour(config.Unavailable) {
			//kjøre reassign??
			//}
			if newState.Floor != elevators[myID].Floor || newState.Behaviour == le.DoorOpen || newState.Behaviour == le.Idle { //Trenger vi å sjekke endring i direction?
				elevators[myID].Floor = newState.Floor
				ch_watchdogPet <- false
			}
			elevators[myID].Behaviour = config.ElevBehaviour(int(newState.Behaviour))
			elevators[myID].Direction = config.MotorDirection(int(newState.Direction))

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
			fmt.Println(elevators[myID])
			Broadcast(elevators, ch_NetworkMessageTx)
			SetHallLights(elevators)

		case updatedElevators := <-ch_NetworkMessageRx:
			//Updates the local map of elevators
			fmt.Println("Updated elevators")
			for ID, elev := range updatedElevators {
				for floor := range elev.Requests {
					for button := range elev.Requests[floor] {
						if _, IDexist := elevators[ID]; !IDexist {
							elevators[ID] = &elev
							fmt.Println("New ID")
						}
						//Når en ny heis kobles på, sliter den med å kopiere den andre heisens ordre
						if !(updatedElevators[myID].Requests[floor][button] == config.Unconfirmed && elevators[myID].Requests[floor][button] == config.Confirmed) {
							elevators[ID].Requests[floor][button] = updatedElevators[ID].Requests[floor][button]
						}
						if elevators[myID].Behaviour != config.Unavailable && elevators[myID].Requests[floor][button] == config.Unconfirmed {
							elevators[myID].Requests[floor][button] = config.Confirmed
							ch_orderToElev <- le.ButtonEvent{floor, le.ButtonType(button)}
							Broadcast(elevators, ch_NetworkMessageTx)
						}
						if elev.Requests[floor][button] == config.Completed {
							elevators[ID].Requests[floor][button] = config.None
							Broadcast(elevators, ch_NetworkMessageTx) //Skal vi flytt broadcast ut av for-loopen?
						}
						SetHallLights(elevators)
					}
				}
				elevators[ID].Floor = updatedElevators[ID].Floor
				elevators[ID].Direction = updatedElevators[ID].Direction
				elevators[ID].Behaviour = updatedElevators[ID].Behaviour
			}
			fmt.Println(elevators[myID])

		// Reassign orders to elevators - uenig

		//case peerUpdate := <-ch_peerUpdate:

		case <-ch_watchdogBark:
			fmt.Println("Watchdog")
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
