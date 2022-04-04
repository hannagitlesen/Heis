package assigner

import (
	"config"
	"elevio"
)

func AssignOrder(elevators map[string]config.DistributorElevator, request elevio.ButtonEvent, myID string) string {
	id := "NoID"
	if len(elevators) == 1 || request.Button == config.BT_Cab { 
		id = myID
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
	return id
}

