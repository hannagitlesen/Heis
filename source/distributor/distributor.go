package distributor

import (
	"assigner"
	"config"
	"elevio"
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

func Broadcast(myID string, msgType config.MessageType, elevators map[string]*config.DistributorElevator, order config.OrderMessage, ch_NetworkMessageTx chan<- config.BroadcastMessage) {
	elevatorsCopy := make(map[string]config.DistributorElevator, 0)
	for id, elev := range elevators {
		elevatorsCopy[id] = *elev
	}

	ch_NetworkMessageTx <- config.BroadcastMessage{SenderID: myID, MsgType: msgType, ElevStatusMsg: elevatorsCopy, OrderMsg: order}

	time.Sleep(time.Millisecond * 50)
}

func SetHallLights(elevators map[string]config.DistributorElevator) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for button := config.BT_HallUp; button <= config.BT_HallDown; button++ {
			lightsOn := false
			for _, elev := range elevators {
				if elev.Requests[floor][button] == config.Confirmed {
					lightsOn = true
				}
			}
			elevio.SetButtonLamp(elevio.ButtonType(button), floor, lightsOn)
		}
	}
}

func Distributor(
	myID string,
	ch_newLocalState chan le.Elevator,
	ch_buttonPress chan elevio.ButtonEvent,
	ch_resetLocalHallOrders chan bool,
	ch_orderToElev chan elevio.ButtonEvent,
	ch_arrivedAtFloors chan int,
	ch_obstr chan bool,
	ch_peerUpdate chan peers.PeerUpdate,
	ch_peerTxEnable chan bool,
	ch_NetworkMessageTx chan config.BroadcastMessage,
	ch_NetworkMessageRx chan config.BroadcastMessage,
	ch_orderFromRemoteElev chan config.OrderMessage,
	ch_watchdogPet chan bool,
	ch_watchdogBark chan bool) {

	elevators := make(map[string]*config.DistributorElevator)
	thisElevator := new(config.DistributorElevator)
	*thisElevator = InitDistributorElev()
	elevators[myID] = thisElevator

	connectTimer := time.NewTimer(time.Duration(config.ConnectTimeout) * time.Second)
	BroadcastStateTimer := time.NewTimer(time.Duration(config.BcastStateTimeout) * time.Millisecond)

	select {
	case initMsgFromNetwork := <-ch_NetworkMessageRx:
		for ID, elev := range initMsgFromNetwork.ElevStatusMsg {
			if ID == myID {
				for floor := range elevators[myID].Requests {
					if elev.Requests[floor][config.BT_Cab] == config.Confirmed || elev.Requests[floor][config.BT_Cab] == config.Unconfirmed {
						ch_buttonPress <- elevio.ButtonEvent{floor, config.BT_Cab}
					}
				}
			} else { //Make sure that new elevator is updated on states
				tempElev := new(config.DistributorElevator)
				*tempElev = elev
				elevators[ID] = tempElev
			}
		}

		order := new(config.OrderMessage)
		Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)

		elevatorsCopy := make(map[string]config.DistributorElevator, 0)
		for id, elev := range elevators {
			elevatorsCopy[id] = *elev
		}
		SetHallLights(elevatorsCopy)
		time.Sleep(time.Second)
		break
	case <-connectTimer.C:
		break
	}

	for {
		select {
		case newLocalOrder := <-ch_buttonPress:
			fmt.Printf("Before assigner:  %v\t: %+v\n", myID, elevators[myID])

			elevatorsCopy := make(map[string]config.DistributorElevator, 0)
			for id, elev := range elevators {
				elevatorsCopy[id] = *elev
			}
			assignedID := assigner.AssignOrder(elevatorsCopy, newLocalOrder, myID) //Pass by value/reference, kan det være denne som sletter ordre?
			if assignedID == myID {
				if !(elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] == config.Confirmed) {
					fmt.Printf("Before newLocalOrder:  %v\t: %+v\n", myID, elevators[myID])
					elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] = config.Unconfirmed
					ch_orderToElev <- newLocalOrder
				}
			} else {
				elevs := make(map[string]*config.DistributorElevator)
				Broadcast(myID, config.MessageType(config.Order), elevs, config.OrderMessage{AssignedID: assignedID, Order: newLocalOrder}, ch_NetworkMessageTx)
			}

		case newState := <-ch_newLocalState:

			fmt.Printf("[distributor] new state:\n     \t %+v\n", newState)
			fmt.Println("[distributor] all elevator states:")
			for k, v := range elevators {
				fmt.Printf("  %v\t: %+v\n", k, v)
			}

			if newState.Floor != elevators[myID].Floor || newState.Behaviour == le.DoorOpen || newState.Behaviour == le.Idle { //Trenger vi å sjekke endring i direction?
				elevators[myID].Floor = newState.Floor
				ch_watchdogPet <- false
			}
			elevators[myID].Behaviour = config.ElevBehaviour(int(newState.Behaviour))
			elevators[myID].Direction = config.MotorDirection(int(newState.Direction))

			for floor := range newState.Requests {
				for button := range newState.Requests[floor] {
					if newState.Requests[floor][button] && elevators[myID].Requests[floor][button] == config.Unconfirmed {
						elevators[myID].Requests[floor][button] = config.Confirmed
						fmt.Printf("[distributor] local confirmed f:%v b:%v\n", floor, button)
						fmt.Printf("After confirmation:  %v\t: %+v\n", myID, elevators[myID])
					}

					if !newState.Requests[floor][button] && elevators[myID].Requests[floor][button] == config.Confirmed {
						fmt.Printf("[distributor] local completed f:%v b:%v\n", floor, button)
						elevators[myID].Requests[floor][button] = config.None
					}
				}
			}

			// order := new(config.OrderMessage)
			// Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)
			//SetHallLights(elevators)

		case msgFromNetwork := <-ch_NetworkMessageRx:

			switch msgFromNetwork.MsgType {
			case config.Order:
				// Må vi sjekke om vi selv er unavailable????
				if msgFromNetwork.OrderMsg.AssignedID == myID {
					if !(elevators[myID].Requests[msgFromNetwork.OrderMsg.Order.Floor][msgFromNetwork.OrderMsg.Order.Button] == config.Confirmed) {
						elevators[myID].Requests[msgFromNetwork.OrderMsg.Order.Floor][msgFromNetwork.OrderMsg.Order.Button] = config.Unconfirmed
						ch_orderToElev <- msgFromNetwork.OrderMsg.Order
					}
				}

			case config.ElevStatus:
				fmt.Printf("new msgfromnetwork:  %v\t: %+v\n", myID, elevators[myID])
				for ID, elev := range msgFromNetwork.ElevStatusMsg {
					if _, IDexist := elevators[ID]; !IDexist {
						tempElev := new(config.DistributorElevator)
						*tempElev = elev
						elevators[ID] = tempElev
					}
				}

				senderID := msgFromNetwork.SenderID

				if senderID != myID {
					if msgFromNetwork.ElevStatusMsg[senderID].Behaviour == config.Unavailable {
						for floor := range msgFromNetwork.ElevStatusMsg[senderID].Requests {
							for button := elevio.BT_HallUp; button <= elevio.BT_HallDown; button++ {

								if msgFromNetwork.ElevStatusMsg[senderID].Requests[floor][button] == config.Unconfirmed || msgFromNetwork.ElevStatusMsg[senderID].Requests[floor][button] == config.Confirmed {
									elevators[myID].Requests[floor][button] = config.Unconfirmed
									ch_orderToElev <- elevio.ButtonEvent{Floor: floor, Button: button}
								}
								elevators[senderID].Requests[floor][button] = config.None // Kan det bli overskriving her?
							} // Reassign????
						}
					} else {
						for floor := range elevators[senderID].Requests {
							for button := range elevators[senderID].Requests[floor] {
								// Må det være en condition her?
								// bug: transition from unconf to none is still allowed by this! that's wrong!
								//fmt.Printf("[distributor] transition (via sender %s) f:%v b:%v to %v\n", senderID, floor, button, msgFromNetwork.ElevStatusMsg[senderID].Requests[floor][button])
								elevators[senderID].Requests[floor][button] = msgFromNetwork.ElevStatusMsg[senderID].Requests[floor][button]
							}
						}
						elevators[senderID].Floor = msgFromNetwork.ElevStatusMsg[senderID].Floor
						elevators[senderID].Direction = msgFromNetwork.ElevStatusMsg[senderID].Direction
						elevators[senderID].Behaviour = msgFromNetwork.ElevStatusMsg[senderID].Behaviour

						fmt.Printf("senderID: %v\n", senderID)
						fmt.Printf("copied from net:  %v\t: %+v\n", myID, elevators[myID])
					}
				}
			}

		case <-BroadcastStateTimer.C:
			order := new(config.OrderMessage)
			Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)
			elevatorsCopy := make(map[string]config.DistributorElevator, 0)
			for id, elev := range elevators {
				elevatorsCopy[id] = *elev
			}
			SetHallLights(elevatorsCopy)
			BroadcastStateTimer.Reset(time.Duration(config.BcastStateTimeout) * time.Millisecond)

		case peerUpdate := <-ch_peerUpdate:
			if len(peerUpdate.Lost) != 0 {
				for _, lostID := range peerUpdate.Lost {
					for ID, elev := range elevators {
						if lostID == ID {
							elev.Behaviour = config.Unavailable // kan det oppstå problem hvis id-en ikke er i dennes elevators

							for floor := range elev.Requests {
								for button := elevio.BT_HallUp; button <= elevio.BT_HallDown; button++ {

									if elev.Requests[floor][button] == config.Unconfirmed || elev.Requests[floor][button] == config.Confirmed {
										elevators[myID].Requests[floor][button] = config.Unconfirmed
										ch_orderToElev <- elevio.ButtonEvent{Floor: floor, Button: button}
									}
									elev.Requests[floor][button] = config.None // Kan det bli overskriving her?
								}
								elev.Requests[floor][elevio.BT_Cab] = config.None // denne må gjøres penere
							}

						}
					}
				}
			}
			//Broadcast?? Nei?

		case <-ch_watchdogBark: //Når den starter igjen her - kommer vi ut av for-loopen?
			fmt.Println("Watchdog")
			elevators[myID].Behaviour = config.Unavailable
			order := new(config.OrderMessage)
			Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)

			for floor := range elevators[myID].Requests { // Er dette nødvendig/ kan det føre til overskriving?
				for button := range elevators[myID].Requests[floor] {
					elevators[myID].Requests[floor][button] = config.None
				}
			}
			ch_resetLocalHallOrders <- true
		}

	}

}

/*
A
on new button
	perform assign
	send to assigned elevator (ONCE!)
on received order from net
	if to self, assign to fsm
on new state from local
	store here in map of all elevators
periodically
	send local state on net
	set lights based on or() of all elevators's orders
on peer loss
	take lost elevator's orders and assign to fsm


B
none -> unconf
	if local button press
	or remote says so
unconf -> conf
	all have acked (who is "all"?)
	or remote says so			for floor := range elevators[myID].Requests { // Er dette nødvendig/ kan det føre til overskriving?
				for button := range elevators[myID].Requests[floor] {
					elevators[myID].Requests[floor][button] = config.None
				}
			}
unkwown -> (any)
none -> unknown
	if we are lone on network

orderstate:
	[unknown, none, unconf, conf]
	assigned id
	ack list


map[string][][]orderstate
	 ^- refers to either "assigned id", or "has acked"
	 	cant be both!


[4][2]struct{orderstate, assignedid, acklist}





*/
