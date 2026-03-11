// This file contains the order handler for the elevator system... tbc

package elevator

import (
	"heis/config"
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
		for flr := floor - 1; flr >= 0; flr-- {
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
