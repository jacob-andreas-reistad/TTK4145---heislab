package main

import (
	"flag"
	"fmt"
	"heis/assigner"
	"heis/config"
	"heis/distributor"
	"heis/elevator"
	"heis/elevio"
	"heis/lights"
	"heis/network/bcast"
	"heis/network/peers"
	"strconv"
)

func main() {

	// Command line arguments for elevator ID and port number:
	elevIDFlag := flag.Int("id", 0, "<-- Default value, override with command line argument -id=x")
	portFlag := flag.Int("port", 15657, "<-- Default value, override with command line argument -port=xxxx")
	flag.Parse()

	id := *elevIDFlag
	port := *portFlag

	// Initialize elevator hardware:
	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors)
	fmt.Println("Initialized Elevator id: ", id, " on port: ", port)
	fmt.Println("This system has ", config.NumFloors, " floors,", config.NumElevators, " elevators and ", config.NumButtons, " buttons per floor.")

	// Channels for communication between goroutines:
	newOrderCh := make(chan elevator.Order, config.Buffer)
	orderDoneCh := make(chan elevio.ButtonEvent, config.Buffer)
	stateUpdateCh := make(chan elevator.State, config.Buffer)
	csConfirmedCh := make(chan distributor.CommonState, config.Buffer)
	networkTx := make(chan distributor.CommonState, config.Buffer)
	networkRx := make(chan distributor.CommonState, config.Buffer)
	peersTx := make(chan bool, config.Buffer)
	peersRx := make(chan peers.PeerUpdate, config.Buffer)

	// Start goroutines:
	go peers.Receiver(config.PeersPortNumber, peersRx)
	go peers.Transmitter(config.PeersPortNumber, strconv.Itoa(id), peersTx)
	go bcast.Receiver(config.BcastPortNumber, networkRx)
	go bcast.Transmitter(config.BcastPortNumber, networkTx)
	go distributor.Synchronizer(id, stateUpdateCh, peersRx, networkTx, networkRx, csConfirmedCh, orderDoneCh)
	go elevator.Elevator(newOrderCh, orderDoneCh, stateUpdateCh, distributor.LoadDirection(id))

	// Main loop: handle confirmed common states, assign new orders and update lights
	for {
		cs := <-csConfirmedCh
		newOrderCh <- assigner.CostFunction(cs, id)
		lights.SetPanelLights(cs, id)
	}
}
