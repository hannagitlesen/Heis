package assigner

import (
	"config"
	le "localelevator"
)


func AssignOrder(elevators map[string]*config.DistributorElevator, request le.ButtonEvent) {
	if len(elevators) == 1 { // single elevator
		for _, elev := range elevators {
			elev.Requests[request.Floor][request.Button] = config.Unconfirmed
			return
		}
	} 
	id := "NoID"
	if request.Button == config.BT_Cab {
		id = config.GetLocalIP()
	} else {
		minCost := 9999
		elevCost := 0
		for elevID, elev := range elevators {
			if elev.Behaviour != config.Unavailable {
				elevCost = TimeToIdle(elev, request)
			} else {
				elevCost = 10000
			}

			if elevCost < minCost {
				id = elevID
				minCost = elevCost
			}
		}
	}
	elevators[id].Requests[request.Floor][request.Button] = config.Unconfirmed
}

func ReassignOrder() {
	// reassign orders if the elevator falls of the network



}
