package elevator

import (
	"heis/config"
	"heis/elevio"
)

// Order represents the state of all orders in the system,
// where each element is true if there is an active order for that floor and button type
type Order [config.NumFloors][config.NumButtons]bool

// hasOrders checks if there are any orders in the given direction from the given floor
func (o Order) hasOrders(direction MotorDirection, floor int) bool {
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

// orderComplete checks if the given order is complete at the given floor and direction,
// and sends a ButtonEvent to the orderDoneCh if it is
func orderComplete(o Order, direction MotorDirection, floor int, orderDoneCh chan<- elevio.ButtonEvent) {
	if o[floor][elevio.BT_Cab] {
		orderDoneCh <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_Cab}
	}
	if o[floor][direction.buttonType()] {
		orderDoneCh <- elevio.ButtonEvent{Floor: floor, Button: direction.buttonType()}
	}
}
