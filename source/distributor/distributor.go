package distributor

import (
	"assigner"
	"config"
	le "localelevator"
	"peers"
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
			} //SOS HJÆÆÆLP

		}
	}

}

func Distributor(
	id string,
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
	*thisElevator = InitDistributorElev(id)
	elevators[id] = thisElevator

	select {
	case newElevators := <-ch_NetworkMessageRx:
		for ID, elev := range newElevators {
			if ID == id {
				for floor := range elevators[id].Requests {
					if elev.Requests[floor][config.BT_Cab] == config.Confirmed || elev.Requests[floor][config.BT_Cab] == config.Unconfirmed {
						//vi sender ut disse ordrene på nytt
					}
				}
			}

		}

		//Sette cab fra matrisen

	}
	for {
		select {
		case newState := <-ch_newLocalState:

		//kjøre reassign??

		case newLocalOrder := <-ch_newLocalOrder:
			assigner.AssignOrder(elevators, newLocalOrder)
			if elevators[id].Requests[newLocalOrder.Floor][newLocalOrder.Button] == config.Unconfirmed {
				Broadcast(elevators, ch_NetworkMessageTx)
				elevators[id].Requests[newLocalOrder.Floor][newLocalOrder.Button] = config.Confirmed
				SetHallLights(elevators)
				ch_orderToElev <- newLocalOrder
			}

			Broadcast(elevators, ch_NetworkMessageTx)
			SetHallLights(elevators)

		case updatedElevators := <-ch_NetworkMessageRx:

		case peerUpdate := <-ch_peerUpdate:

		case <-ch_watchdogBark:
			elevators[id].Behaviour = config.Unavailable
			Broadcast(elevators, ch_NetworkMessageTx)
			for floor := range elevators[id].Requests {
				for button := range elevators[id].Requests[floor] {
					elevators[id].Requests[floor][button] = config.None
				}
			}

		}

	}

}
