package distributor

import (
	"assigner"
	"config"
	"fmt"
	le "localelevator"
	"peers"
	"time"
)

//Ahhhhhhhhhh
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
	ch_clearLocalHallOrders chan bool,
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
		SetHallLights(elevators)
		fmt.Println("before break")
		time.Sleep(time.Second)
		break
	case <-connectTimer.C:
		fmt.Println("connect timer")
		break
	}

	for {
		select {
		case newLocalOrder := <-ch_newLocalOrder:
			assigner.AssignOrder(elevators, newLocalOrder, myID)
			fmt.Println("new local order")
			if elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] == config.Unconfirmed {
				fmt.Println("My order")
				Broadcast(elevators, ch_NetworkMessageTx)
				fmt.Println("before confirmed")
				elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] = config.Confirmed
				fmt.Println("before lights")
				SetHallLights(elevators)
				ch_orderToElev <- newLocalOrder
				fmt.Println("given order to fsm")
			}
			fmt.Println("before broadcast")
			SetHallLights(elevators)
			Broadcast(elevators, ch_NetworkMessageTx)

		case newState := <-ch_newLocalState:
			fmt.Println("New local state")
			fmt.Println(newState)
			fmt.Println(elevators[myID])

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
			SetHallLights(elevators)
			fmt.Println(elevators[myID])
			Broadcast(elevators, ch_NetworkMessageTx)

		case updatedElevators := <-ch_NetworkMessageRx:
			//Updates the local map of elevators
			fmt.Println("Updated elevators")
			for ID, elev := range updatedElevators {
				if elev.Behaviour == config.Unavailable {
					assigner.ReassignOrder(elevators, ch_newLocalOrder, myID)
					for floor := range elev.Requests {
						for button := config.BT_HallUp; button <= config.BT_HallDown; button++ {
							elevators[ID].Requests[floor][button] = config.None
						}
					}
					SetHallLights(elevators)
				}

				for floor := range elev.Requests {
					for button := range elev.Requests[floor] {
						if _, IDexist := elevators[ID]; !IDexist {
							elevators[ID] = &elev
							fmt.Println("New ID")
						}
						if len(updatedElevators) > 2 {
							panic("")
						}
						if len(elevators) > 2 {
							panic("")
						}
						//Når en ny heis kobles på, sliter den med å kopiere den andre heisens ordre
						if !(updatedElevators[myID].Requests[floor][button] == config.Unconfirmed && elevators[myID].Requests[floor][button] == config.Confirmed) {
							elevators[ID].Requests[floor][button] = updatedElevators[ID].Requests[floor][button]
						}
						if elevators[myID].Behaviour != config.Unavailable && elevators[myID].Requests[floor][button] == config.Unconfirmed {
							elevators[myID].Requests[floor][button] = config.Confirmed
							fmt.Println("Confirm")
							ch_orderToElev <- le.ButtonEvent{floor, le.ButtonType(button)}
							SetHallLights(elevators)
							Broadcast(elevators, ch_NetworkMessageTx)
						}
						if elev.Requests[floor][button] == config.Completed {
							fmt.Println("Remove orders")
							elevators[ID].Requests[floor][button] = config.None
							Broadcast(elevators, ch_NetworkMessageTx)
						}
					}
				}
				elevators[ID].Floor = updatedElevators[ID].Floor
				elevators[ID].Direction = updatedElevators[ID].Direction
				elevators[ID].Behaviour = updatedElevators[ID].Behaviour
			}
			SetHallLights(elevators)
			fmt.Println(elevators[myID])

		case peerUpdate := <-ch_peerUpdate:
			if len(peerUpdate.Lost) != 0 {
				for _, lostID := range peerUpdate.Lost {
					for ID, elev := range elevators {
						if lostID == ID {
							elev.Behaviour = config.Unavailable // kan det oppstå problem hvis id-en ikke er i dennes elevators
						}
					}
				}
				Broadcast(elevators, ch_NetworkMessageTx)
			}

		case <-ch_watchdogBark: //Når den starter igjen her - kommer vi ut av for-loopen?
			fmt.Println("Watchdog")
			elevators[myID].Behaviour = config.Unavailable
			Broadcast(elevators, ch_NetworkMessageTx)
			ch_clearLocalHallOrders <- true
		}

	}

}
