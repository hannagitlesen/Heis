package assigner

import (
	"config"
	"fmt"
	le "localelevator"
)

func AssignOrder(elevators map[string]*config.DistributorElevator, request le.ButtonEvent, myID string) {
	if len(elevators) == 1 { // single elevator
		for _, elev := range elevators {
			elev.Requests[request.Floor][request.Button] = config.Unconfirmed
			return
		}
	}
	id := "NoID"
	if request.Button == config.BT_Cab {
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
	elevators[id].Requests[request.Floor][request.Button] = config.Unconfirmed
}

func ReassignOrder(elevators map[string]*config.DistributorElevator, ch_newLocalOrder chan le.ButtonEvent, myID string) {
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
							ch_newLocalOrder <- le.ButtonEvent{floor, le.ButtonType(button)}
							fmt.Println("sent local order")
						}
					}
				}
			}
		}
	}
}
