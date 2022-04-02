package main

import (
	"config"
	le "localelevator"
	"elevio"

	//	"assigner"
	"bcast"
	//"conn"
	//"localip"
	"flag"
	"peers"

	//"os"
	"distributor"
	"watchdog"
)

func main() {
	//e := le.NewElevator()
	//fmt.Printf("%+v\n", e)

	/*fmt.Println("Started")
	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

		case a := <-helloRx:
			fmt.Printf("Received: %#v\n", a)
		}
	}*/

	//go run main.go -port=our_port -id=our_id
	var port string
	var id string
	flag.StringVar(&port, "port", "", "port of this peer")
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()
	// if true {
	// 	panic("")
	// }

	elevio.Init("localhost:"+port, config.NumFloors)

	//Channels for communication between distributor and local elevator
	ch_newLocalState := make(chan le.Elevator)
	ch_orderToFSM := make(chan elevio.ButtonEvent, 100)
	ch_buttonPress := make(chan elevio.ButtonEvent, 100)
	ch_resetLocalHallOrders := make(chan bool)

	//Channels for communication between local elevator and elevio
	ch_arrivedAtFloors := make(chan int)
	ch_obstr := make(chan bool)

	//Channels for communication between distributor and network
	ch_peerUpdate := make(chan peers.PeerUpdate)
	ch_peerTxEnable := make(chan bool)
	ch_NetworkMessageTx := make(chan config.BroadcastMessage)
	ch_NetworkMessageRx := make(chan config.BroadcastMessage)
	ch_orderFromRemoteElev := make(chan config.OrderMessage)

	//Channels for communication between distributor and watchdog
	ch_watchdogPet := make(chan bool)
	ch_watchdogBark := make(chan bool)

	//Goroutines for local elevator
	go elevio.PollButtons(ch_buttonPress)
	go elevio.PollFloorSensor(ch_arrivedAtFloors)
	go elevio.PollObstructionSwitch(ch_obstr)

	go le.FSM(ch_newLocalState, ch_orderToFSM, ch_resetLocalHallOrders, ch_arrivedAtFloors, ch_obstr)

	//Goroutines for network
	go peers.Transmitter(config.PeersPort, id, ch_peerTxEnable)
	go peers.Receiver(config.PeersPort, ch_peerUpdate)
	go bcast.Transmitter(config.BcastPort, ch_NetworkMessageTx)
	go bcast.Receiver(config.BcastPort, ch_NetworkMessageRx)

	//Goroutine for watchdog
	go watchdog.Watchdog(config.WatchdogTimeout, ch_watchdogPet, ch_watchdogBark)

	//Goroutine for distributor
	go distributor.Distributor(
		id,
		ch_newLocalState,
		ch_buttonPress,
		ch_resetLocalHallOrders,
		ch_orderToFSM,
		ch_arrivedAtFloors,
		ch_obstr,
		ch_peerUpdate,
		ch_peerTxEnable,
		ch_NetworkMessageTx,
		ch_NetworkMessageRx,
		ch_orderFromRemoteElev,
		ch_watchdogPet,
		ch_watchdogBark)

	select {}
}
