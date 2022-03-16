package main

import (
	le "localelevator"
	"config"
//	"assigner"
//	"network"
)

func main() {
	//e := le.NewElevator()
	//fmt.Printf("%+v\n", e)
	le.Init("localhost:15657", config.NumFloors)

	ch_newLocalState := make(chan le.ElevBehaviour) //buffer channels?
	ch_orderToElev := make(chan le.ButtonEvent)
	ch_arrivedAtFloors := make(chan int)
	ch_obstr := make(chan bool)

	go le.PollButtons(ch_orderToElev)
	go le.PollFloorSensor(ch_arrivedAtFloors)
	go le.PollObstructionSwitch(ch_obstr)

	go le.FSM(ch_newLocalState, ch_orderToElev, ch_arrivedAtFloors, ch_obstr)

	select {}
}
