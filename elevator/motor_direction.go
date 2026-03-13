// This file contains the motor direction control for the elevator system... tbc

package elevator

import "heis/elevio"

type MotorDirection int

const (
	Stop MotorDirection = 0
	Up   MotorDirection = 1
	Down MotorDirection = -1
)


func (md MotorDirection) motor_direction() elevio.MotorDirection {
	switch md {
	case Up:
		return elevio.MD_Up
	case Down:
		return elevio.MD_Down
	default:
		return elevio.MD_Stop
	}
}

func (md MotorDirection) button_type() elevio.ButtonType {
	switch md {
	case Up:
		return elevio.BT_HallUp
	case Down:
		return elevio.BT_HallDown
	default:
		return elevio.BT_Cab
	}
}

func (md MotorDirection) opposite() MotorDirection {
	switch md {
	case Up:
		return Down
	case Down:
		return Up
	default:
		return md
	}
}

func (md MotorDirection) ToString() string {
	switch md {
	case Up:
		return "Up"
	case Down:
		return "Down"
	default:
		return "Stop"
	}
}
