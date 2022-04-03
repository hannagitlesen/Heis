package main

import (
	"config"
	localElev "localelevator"
	"elevio"
	"flag"
	"distributor"
)

func main() {
	//go run main.go -port=our_port -id=our_id
	var port string
	var id string
	flag.StringVar(&port, "port", "", "port of this peer")
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	elevio.Init("localhost:"+port, config.NumFloors)

	//Channels for communication between distributor and local elevator
	ch_newLocalState := make(chan localElev.Elevator)
	ch_orderToFSM := make(chan elevio.ButtonEvent, 100)
	ch_buttonPress := make(chan elevio.ButtonEvent, 100)
	ch_resetLocalHallOrders := make(chan bool)

	//Channels for communication between local elevator and elevio
	ch_arrivedAtFloors := make(chan int)
	ch_obstr := make(chan bool)



	//Goroutines for local elevator
	go elevio.PollButtons(ch_buttonPress)
	go elevio.PollFloorSensor(ch_arrivedAtFloors)
	go elevio.PollObstructionSwitch(ch_obstr)

	go localElev.FSM(ch_newLocalState, ch_orderToFSM, ch_resetLocalHallOrders, ch_arrivedAtFloors, ch_obstr)

	//Goroutine for distributor
	go distributor.Distributor(
		id,
		ch_newLocalState,
		ch_buttonPress,
		ch_resetLocalHallOrders,
		ch_orderToFSM,
		ch_arrivedAtFloors,
		ch_obstr,
	)

	select {}
}
