package assigner

import (
	"config"
	"elevio"
	"fmt"
)

func AssignOrder(elevators map[string]config.DistributorElevator, request elevio.ButtonEvent, myID string) string {
	id := "NoID"
	if len(elevators) == 1 || request.Button == config.BT_Cab { // single elevator
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

func ReassignOrder(elevators map[string]*config.DistributorElevator, ch_newLocalOrder chan elevio.ButtonEvent, myID string) {
	fmt.Println("reassign")
	highestID := ""
	for ID, elev := range elevators {
		if elev.Behaviour != config.Unavailable {
			if ID > highestID {
				highestID = ID
			}
		}
	}
	fmt.Println(highestID)
	fmt.Println(myID)
	for _, elev := range elevators {
		if elev.Behaviour == config.Unavailable {
			for floor := range elev.Requests {
				for button := config.BT_HallUp; button <= config.BT_HallDown; button++ {
					if elev.Requests[floor][button] == config.Unconfirmed || elev.Requests[floor][button] == config.Confirmed {
						if highestID == myID {
							ch_newLocalOrder <- elevio.ButtonEvent{floor, elevio.ButtonType(button)}
							fmt.Println("sent local order")
						}
					}
				}
			}
		}
	}
}
