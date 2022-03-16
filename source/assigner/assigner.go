package assigner

import (
	"config"
)

func AssignOrder(elevators []*config.DistributorElevator, request config.ButtonEvent) {
	if len(elevators) == 1 || request.Button == config.BT_Cab { // single elevator
		elevators[0].Requests[request.Floor][request.Button] = config.Unconfirmed
	}

	var minElev *config.DistributorElevator
	minCost := 9999
	elevCost := 0
	for _, elev := range elevators {
		if elev.Behaviour != config.Unavailable {
			elevCost = TimeToIdle(elev, request)
		} else {
			elevCost = 10000
		}

		if elevCost < minCost {
			minElev = elev
			minCost = elevCost
		}
	}
	minElev.Requests[request.Floor][request.Button] = config.Unconfirmed
}

func ReassignOrder() {
	// reassign orders if the elevator falls of the network

}
