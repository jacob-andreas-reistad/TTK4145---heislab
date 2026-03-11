// This file contains the order handler for the elevator system... tbc

package elevator

import (
	"heis/config"
	"heis/elevio"
)

type Order [config.NumFloors][config.NumButtons]bool

func (o Order) has_orders(direction MotorDirection, floor int) bool {
	switch direction {

	case Up:
		for flr := floor + 1; flr < config.NumFloors; flr++ {
			for btn := 0; btn < config.NumButtons; btn++ {
				if o[flr][btn] {
					return true
				}
			}
		}
		return false

	case Down:
		for flr := 0; flr < floor; flr++ {
			for btn := 0; btn < config.NumButtons; btn++ {
				if o[flr][btn] {
					return true
				}
			}
		}
		return false
	default:
		panic("invalid direction")
	}
}

func order_complete(o Order, direction MotorDirection, floor int, orderDoneCh chan<- elevio.ButtonEvent) {
	if o[floor][elevio.BT_Cab] {
		orderDoneCh <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_Cab}
	}
	if o[floor][direction] {
		orderDoneCh <- elevio.ButtonEvent{Floor: floor, Button: direction.button_type()}
	}
}
