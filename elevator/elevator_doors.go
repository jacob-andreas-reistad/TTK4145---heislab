// This file contains the FSM for the elevator system... tbc

package elevator

import (
	"heis/config"
	"heis/elevio"
)

type Orders [config.NumFloors][config.NumButtons]bool

func (o Orders) has_order(dir MotorDirection, floor int) bool {
	// Check if there is an order in the given direction at the given floor
}

func order_done(dir MotorDirection, floor int, o Orders, orderDoneCh chan<- elevio.ButtonEvent){
	// Send an order done event to the order handler
}