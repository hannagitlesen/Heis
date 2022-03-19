package main

import (
	"config"
	le "localelevator"

	//	"assigner"
	"bcast"
	//"conn"
	//"localip"
	"flag"
	"peers"

	//"os"
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
	flag.StringVar(&port, "port", "", "port of this peer")
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()
	
	le.Init("localhost:"+port, config.NumFloors)

	//Channels for communication between distributor and local elevator	
	ch_newLocalState := make(chan le.ElevBehaviour)
	ch_orderToElev := make(chan le.ButtonEvent)

	//Channels for communication between local elevator and elevio
	ch_arrivedAtFloors := make(chan int)
	ch_obstr := make(chan bool)

	//Channels for communication between distributor and network
	ch_peerUpdate := make(chan peers.PeerUpdate)
	ch_peerTxEnable := make(chan bool)
	ch_NetworkMessageTx := make(chan []config.DistributorElevator)
	ch_NetworkMessageRx := make(chan []config.DistributorElevator)

	//Channels for communication between distributor and watchdog
	ch_watchdogPet := make(chan bool)
	ch_watchdogBark := make(chan bool)

	//Goroutines for local elevator
	go le.PollButtons(ch_orderToElev)
	go le.PollFloorSensor(ch_arrivedAtFloors)
	go le.PollObstructionSwitch(ch_obstr)

	go le.FSM(ch_newLocalState, ch_orderToElev, ch_arrivedAtFloors, ch_obstr)

	//Goroutines for network
	go peers.Transmitter(config.PeersPort, id, ch_peerTxEnable)
	go peers.Receiver(config.PeersPort, ch_peerUpdate)
	go bcast.Transmitter(config.BcastPort, ch_NetworkMessageTx)
	go bcast.Receiver(config.BcastPort, ch_NetworkMessageRx)

	//Goroutine for watchdog
	go watchdog.Watchdog(config.FailureTimeout, ch_watchdogPet, ch_watchdogBark)

	select {}
}
