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

var id int
var Port int

func main() {
	
	elevID := flag.Int("id", 0, "<-- Default value, override with command line argument -id=x")
	port   := flag.Int("port", 15657, "<-- Default value, override with command line argument -port=xxxx")
	flag.Parse()

	id = *elevID
	Port = *port

	elevio.Init("localhost:"+strconv.Itoa(Port), config.NumFloors)

	fmt.Println("Initialized Elevator id: ", id, " on port: ", Port)
	fmt.Println("This system has ", config.NumFloors, " floors,",config.NumElevators, " elevators and ", config.NumButtons, " buttons per floor.")

	// Channels for communication between goroutines
	newOrderCh 		:= make(chan elevator.Order, config.Buffer)
	orderDoneCh 	:= make(chan elevio.ButtonEvent, config.Buffer)
	stateUpdateCh 	:= make(chan elevator.State, config.Buffer)
	CsConfirmedCh 	:= make(chan distributor.CommonState, config.Buffer)
	networkTx 		:= make(chan distributor.CommonState, config.Buffer)
	networkRx 		:= make(chan distributor.CommonState, config.Buffer)
	peersTx 		:= make(chan bool, config.Buffer)
	peersRx 		:= make(chan peers.PeerUpdate, config.Buffer)

	// Start goroutines
	go peers.Receiver(config.PeersPortNumber, peersRx)
	go peers.Transmitter(config.PeersPortNumber, strconv.Itoa(id), peersTx)

	go bcast.Receiver(config.BcastPortNumber, networkRx)
	go bcast.Transmitter(config.BcastPortNumber, networkTx)

	go distributor.Synchronizer(id, stateUpdateCh, peersRx, networkTx, networkRx, CsConfirmedCh, orderDoneCh)

	go elevator.Elevator(newOrderCh, orderDoneCh, stateUpdateCh)


	for {
		select {
		case cs := <- CsConfirmedCh:
			newOrderCh <- assigner.CostFunction(cs, id)
			setlights.SetPanelLights(cs, id)
		default:
			continue
		}
	}
}


	/*
	   elevio.Init("localhost:15657", numFloors)

	   var d elevio.MotorDirection = elevio.MD_Up
	   //elevio.SetMotorDirection(d)

	   drv_buttons := make(chan elevio.ButtonEvent)
	   drv_floors  := make(chan int)
	   drv_obstr   := make(chan bool)
	   drv_stop    := make(chan bool)

	   go elevio.PollButtons(drv_buttons)
	   go elevio.PollFloorSensor(drv_floors)
	   go elevio.PollObstructionSwitch(drv_obstr)
	   go elevio.PollStopButton(drv_stop)


	   for {
	       select {
	       case a := <- drv_buttons:
	           fmt.Printf("%+v\n", a)
	           elevio.SetButtonLamp(a.Button, a.Floor, true)

	       case a := <- drv_floors:
	           fmt.Printf("%+v\n", a)
	           if a == numFloors-1 {
	               d = elevio.MD_Down
	           } else if a == 0 {
	               d = elevio.MD_Up
	           }
	           elevio.SetMotorDirection(d)


	       case a := <- drv_obstr:
	           fmt.Printf("%+v\n", a)
	           if a {
	               elevio.SetMotorDirection(elevio.MD_Stop)
	           } else {
	               elevio.SetMotorDirection(d)
	           }

	       case a := <- drv_stop:
	           fmt.Printf("%+v\n", a)
	           for f := 0; f < numFloors; f++ {
	               for b := elevio.ButtonType(0); b < 3; b++ {
	                   elevio.SetButtonLamp(b, f, false)
	               }
	           }
	       }
	   }*/


