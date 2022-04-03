package distributor

import (
	"assigner"
	"bcast"
	"config"
	"elevio"
	"fmt"
	localElev "localelevator"
	"peers"
	"time"
	"watchdog"
)

func InitDistributorElev() config.DistributorElevator {
	requests := make([][]config.RequestsState, config.NumFloors)
	for floor := range requests {
		requests[floor] = make([]config.RequestsState, config.NumButtons)
	}

	return config.DistributorElevator{Requests: requests, Floor: 0, Behaviour: config.Idle, Direction: config.MD_Stop}
}

func DeepCopyElev(elev config.DistributorElevator) config.DistributorElevator {
	elevCopy := InitDistributorElev()
	elevCopy.Behaviour = elev.Behaviour
	elevCopy.Direction = elev.Direction
	elevCopy.Floor = elev.Floor

	elevCopy.Requests = make([][]config.RequestsState, config.NumFloors)
	for floor := range elevCopy.Requests {
		elevCopy.Requests[floor] = make([]config.RequestsState, config.NumButtons)
		for button := range elevCopy.Requests[floor] {
			elevCopy.Requests[floor][button] = elev.Requests[floor][button]
		}
	}
	return elevCopy
}

func DeepCopyElevMap(elevators map[string]*config.DistributorElevator) map[string]config.DistributorElevator {
	mapCopy := make(map[string]config.DistributorElevator)

	for ID, elev := range elevators {
		mapCopy[ID] = DeepCopyElev(*elev)
	}
	return mapCopy
}

func Broadcast(myID string, msgType config.MessageType, elevators map[string]*config.DistributorElevator, order config.OrderMessage, ch_NetworkMessageTx chan<- config.BroadcastMessage) {
	elevatorsCopy := DeepCopyElevMap(elevators)
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
	ch_newLocalState chan localElev.Elevator,
	ch_buttonPress chan elevio.ButtonEvent,
	ch_resetLocalHallOrders chan bool,
	ch_orderToElev chan elevio.ButtonEvent,
	ch_arrivedAtFloors chan int,
	ch_obstr chan bool,
) {

	//Channels for communication between distributor and network
	ch_peerUpdate := make(chan peers.PeerUpdate)
	ch_peerTxEnable := make(chan bool)
	ch_NetworkMessageTx := make(chan config.BroadcastMessage)
	ch_NetworkMessageRx := make(chan config.BroadcastMessage)

	//Channels for communication between distributor and watchdog
	ch_watchdogPet := make(chan bool)
	ch_watchdogBark := make(chan bool)

	//Goroutines for network
	go peers.Transmitter(config.PeersPort, myID, ch_peerTxEnable)
	go peers.Receiver(config.PeersPort, ch_peerUpdate)
	go bcast.Transmitter(config.BcastPort, ch_NetworkMessageTx)
	go bcast.Receiver(config.BcastPort, ch_NetworkMessageRx)

	//Goroutine for watchdog
	go watchdog.Watchdog(config.WatchdogTimeout, ch_watchdogPet, ch_watchdogBark)

	elevators := make(map[string]*config.DistributorElevator)
	thisElevator := new(config.DistributorElevator)
	*thisElevator = InitDistributorElev()
	elevators[myID] = thisElevator

	connectTimer := time.NewTimer(time.Duration(config.ConnectTimeout) * time.Second)
	BroadcastStateTimer := time.NewTimer(time.Duration(config.BcastStateTimeout) * time.Millisecond)

	select {
	case initMsgFromNetwork := <-ch_NetworkMessageRx:
		if initMsgFromNetwork.MsgType == config.ElevStatus {
			for ID, elev := range initMsgFromNetwork.ElevStatusMsg {
				if ID == myID {
					for floor := range elevators[myID].Requests {
						if elev.Requests[floor][config.BT_Cab] == config.Confirmed || elev.Requests[floor][config.BT_Cab] == config.Unconfirmed {
							ch_buttonPress <- elevio.ButtonEvent{floor, config.BT_Cab}
						}
					}
				} else {
					tempElev := DeepCopyElev(elev)
					elevators[ID] = &tempElev
				}
			}

			order := new(config.OrderMessage)
			Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)

			elevatorsCopy := DeepCopyElevMap(elevators)
			SetHallLights(elevatorsCopy)
			time.Sleep(time.Second)
			break
		}
	case <-connectTimer.C:
		break
	}

	for {
		select {
		case newLocalOrder := <-ch_buttonPress:
			fmt.Printf("Before assigner:  %v\t: %+v\n", myID, elevators[myID])

			elevatorsCopy := DeepCopyElevMap(elevators)
			assignedID := assigner.AssignOrder(elevatorsCopy, newLocalOrder, myID)

			fmt.Printf("After assigner:  %v\t: %+v\n", myID, elevators[myID])

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

			if newState.Floor != elevators[myID].Floor || newState.Behaviour == localElev.DoorOpen || newState.Behaviour == localElev.Idle { //Hva med motorstopp i idle/dooropen? er det viktig?
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

		case msgFromNetwork := <-ch_NetworkMessageRx:

			switch msgFromNetwork.MsgType {
			case config.Order:
				newOrder := msgFromNetwork.OrderMsg.Order
				fmt.Printf("New order, sender ID:  %v\n", msgFromNetwork.SenderID)
				if msgFromNetwork.OrderMsg.AssignedID == myID {
					if !(elevators[myID].Requests[newOrder.Floor][newOrder.Button] == config.Confirmed) {
						elevators[myID].Requests[newOrder.Floor][newOrder.Button] = config.Unconfirmed
						ch_orderToElev <- newOrder
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
								elevators[senderID].Requests[floor][button] = config.None
							}
						}
					} else {
						for floor := range elevators[senderID].Requests {
							for button := range elevators[senderID].Requests[floor] {
								// bug: transition from unconf to none is still allowed by this! that's wrong!
								//fmt.Printf("[distributor] transition (via sender %s) f:%v b:%v to %v\n", senderID, floor, button, msgFromNetwork.ElevStatusMsg[senderID].Requests[floor][button])
								elevators[senderID].Requests[floor][button] = msgFromNetwork.ElevStatusMsg[senderID].Requests[floor][button]
							}
						}
						elevators[senderID].Floor = msgFromNetwork.ElevStatusMsg[senderID].Floor
						elevators[senderID].Direction = msgFromNetwork.ElevStatusMsg[senderID].Direction
						elevators[senderID].Behaviour = msgFromNetwork.ElevStatusMsg[senderID].Behaviour

						fmt.Printf("senderID: %v\n", senderID)
						fmt.Printf("copied from net:  %v\t: %+v\n", senderID, elevators[senderID])
					}
				}
			}

		case <-BroadcastStateTimer.C:
			order := new(config.OrderMessage)
			Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)
			elevatorsCopy := DeepCopyElevMap(elevators)
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
								//Fjerne cab ordre? virker ikke sånn
							}
						}
					}
				}
			}

		case <-ch_watchdogBark:
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
