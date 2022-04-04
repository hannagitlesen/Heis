package distributor

import (
	"assigner"
	"bcast"
	"config"
	"elevio"
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
	ch_NetworkMessageTx <- config.BroadcastMessage{SenderID: myID, MsgType: msgType, ElevsStatusMsg: elevatorsCopy, OrderMsg: order}

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
	BroadcastStateTimer := time.NewTimer(time.Duration(config.BcastStateUpdate) * time.Millisecond)

	select {
	case initMsgFromNetwork := <-ch_NetworkMessageRx:
		if initMsgFromNetwork.MsgType == config.ElevStatus {
			for ID, elev := range initMsgFromNetwork.ElevsStatusMsg {
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
			elevatorsCopy := DeepCopyElevMap(elevators)
			assignedID := assigner.AssignOrder(elevatorsCopy, newLocalOrder, myID)
			if assignedID == myID {
				if !(elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] == config.Confirmed) {
					elevators[myID].Requests[newLocalOrder.Floor][newLocalOrder.Button] = config.Unconfirmed
					ch_orderToElev <- newLocalOrder
				}
			} else {
				elevs := make(map[string]*config.DistributorElevator)
				Broadcast(myID, config.MessageType(config.Order), elevs, config.OrderMessage{AssignedID: assignedID, Order: newLocalOrder}, ch_NetworkMessageTx)
			}

		case newState := <-ch_newLocalState:
			if newState.Floor != elevators[myID].Floor || newState.Behaviour == localElev.DoorOpen || newState.Behaviour == localElev.Idle { 
				elevators[myID].Floor = newState.Floor
				if !(newState.Obstructed && newState.Behaviour == localElev.DoorOpen) {
					ch_watchdogPet <- false
				}
			}
			elevators[myID].Behaviour = config.ElevBehaviour(int(newState.Behaviour))
			elevators[myID].Direction = config.MotorDirection(int(newState.Direction))

			for floor := range newState.Requests {
				for button := range newState.Requests[floor] {
					if newState.Requests[floor][button] && elevators[myID].Requests[floor][button] == config.Unconfirmed {
						elevators[myID].Requests[floor][button] = config.Confirmed
					}

					if !newState.Requests[floor][button] && elevators[myID].Requests[floor][button] == config.Confirmed {
						elevators[myID].Requests[floor][button] = config.None
					}
				}
			}

		case msgFromNetwork := <-ch_NetworkMessageRx:

			switch msgFromNetwork.MsgType {
			case config.Order:
				newOrder := msgFromNetwork.OrderMsg.Order
				if msgFromNetwork.OrderMsg.AssignedID == myID {
					if !(elevators[myID].Requests[newOrder.Floor][newOrder.Button] == config.Confirmed) {
						elevators[myID].Requests[newOrder.Floor][newOrder.Button] = config.Unconfirmed
						ch_orderToElev <- newOrder
					}
				}

			case config.ElevStatus:
				for ID, elev := range msgFromNetwork.ElevsStatusMsg {
					if _, IDexist := elevators[ID]; !IDexist {
						tempElev := new(config.DistributorElevator)
						*tempElev = elev
						elevators[ID] = tempElev
					}
				}

				senderID := msgFromNetwork.SenderID

				if senderID != myID {
					if msgFromNetwork.ElevsStatusMsg[senderID].Behaviour == config.Unavailable {
						for floor := range msgFromNetwork.ElevsStatusMsg[senderID].Requests {
							for button := elevio.BT_HallUp; button <= elevio.BT_HallDown; button++ {

								if msgFromNetwork.ElevsStatusMsg[senderID].Requests[floor][button] == config.Unconfirmed || msgFromNetwork.ElevsStatusMsg[senderID].Requests[floor][button] == config.Confirmed {
									elevators[myID].Requests[floor][button] = config.Unconfirmed
									ch_orderToElev <- elevio.ButtonEvent{Floor: floor, Button: button}
								}
								elevators[senderID].Requests[floor][button] = config.None
							}
						}
					} else {
						for floor := range elevators[senderID].Requests {
							for button := range elevators[senderID].Requests[floor] {
								elevators[senderID].Requests[floor][button] = msgFromNetwork.ElevsStatusMsg[senderID].Requests[floor][button]
							}
						}
						elevators[senderID].Floor = msgFromNetwork.ElevsStatusMsg[senderID].Floor
						elevators[senderID].Direction = msgFromNetwork.ElevsStatusMsg[senderID].Direction
						elevators[senderID].Behaviour = msgFromNetwork.ElevsStatusMsg[senderID].Behaviour
					}
				}
			}

		case <-BroadcastStateTimer.C:
			order := new(config.OrderMessage)
			Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)
			elevatorsCopy := DeepCopyElevMap(elevators)
			SetHallLights(elevatorsCopy)
			BroadcastStateTimer.Reset(time.Duration(config.BcastStateUpdate) * time.Millisecond)

		case peerUpdate := <-ch_peerUpdate:
			if len(peerUpdate.Lost) != 0 {
				for _, lostID := range peerUpdate.Lost {
					for ID, elev := range elevators {
						if lostID == ID {
							elev.Behaviour = config.Unavailable 

							for floor := range elev.Requests {
								for button := elevio.BT_HallUp; button <= elevio.BT_HallDown; button++ {

									if elev.Requests[floor][button] == config.Unconfirmed || elev.Requests[floor][button] == config.Confirmed {
										elevators[myID].Requests[floor][button] = config.Unconfirmed
										ch_orderToElev <- elevio.ButtonEvent{Floor: floor, Button: button}
									}
									elev.Requests[floor][button] = config.None
								}
							}
						}
					}
				}
			}

		case <-ch_watchdogBark:
			elevators[myID].Behaviour = config.Unavailable
			order := new(config.OrderMessage)
			Broadcast(myID, config.MessageType(config.ElevStatus), elevators, *order, ch_NetworkMessageTx)

			for floor := range elevators[myID].Requests {
				for button := elevio.BT_HallUp; button <= elevio.BT_HallDown; button++ {
					elevators[myID].Requests[floor][button] = config.None
				}
			}
			ch_resetLocalHallOrders <- true
		}

	}

}

