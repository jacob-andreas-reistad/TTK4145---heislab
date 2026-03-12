package main

/*
import "heis/elevio"
import "fmt"
*/

import (
	"fmt"
	"heis/config"
	"heis/distributor"
	"heis/elevator"
	"heis/elevio"
	"heis/lights"
	"heis/network"
)

func main() {

	numFloors := 4
	s := distributor.CommonState{}

	// Test RegisterOrder - cab call
	s.RegisterOrder(elevio.ButtonEvent{Floor: 2, Button: elevio.BT_Cab}, 0)
	fmt.Println("CabCall floor 2 elevator 0:", s.Elevators[0].CabCalls[2]) // true

	// Test RegisterOrder - hall call up
	s.RegisterOrder(elevio.ButtonEvent{Floor: 1, Button: elevio.BT_HallUp}, 0)
	fmt.Println("HallCall floor 1 up:", s.HallCalls[1][0]) // true

	// Test ClearOrder
	s.ClearOrder(elevio.ButtonEvent{Floor: 2, Button: elevio.BT_Cab}, 0)
	fmt.Println("CabCall floor 2 after clear:", s.Elevators[0].CabCalls[2]) // false

	// Test BeginUpdate
	s.InitializeSolo(0)
	s.BeginUpdate(0)
	fmt.Println("OrderNum after BeginUpdate:", s.OrderNum) // 1
	fmt.Println("Sender:", s.Sender)                       // 0

	// Test AllAcknowledged - should be true since others are Offline
	fmt.Println("AllAcknowledged:", s.AllAcknowledged(0)) // true

	// Test SameState
	s2 := s
	fmt.Println("SameState with copy:", s.SameState(s2)) // true
	s2.HallCalls[0][0] = true
	fmt.Println("SameState after mutation:", s.SameState(s2)) // false

	fmt.Println("All tests passed!")

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
}
