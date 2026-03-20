package elevator

import "heis/elevio"

// MotorDirection represents the direction of the elevator motor, which can be Up, Down or Stop
type MotorDirection int

const (
	Stop MotorDirection = 0
	Up   MotorDirection = 1
	Down MotorDirection = -1
)

// motorDirection converts the MotorDirection to the corresponding elevio.MotorDirection
func (md MotorDirection) motorDirection() elevio.MotorDirection {
	switch md {
	case Up:
		return elevio.MD_Up
	case Down:
		return elevio.MD_Down
	default:
		return elevio.MD_Stop
	}
}

// buttonType returns the corresponding elevio.ButtonType for the given MotorDirection
func (md MotorDirection) buttonType() elevio.ButtonType {
	switch md {
	case Up:
		return elevio.BT_HallUp
	case Down:
		return elevio.BT_HallDown
	default:
		return elevio.BT_Cab
	}
}

// opposite returns the opposite direction of the given MotorDirection
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

// ToString returns the string representation of the given MotorDirection
func (md MotorDirection) ToString() string {
	switch md {
	case Up:
		return "up"
	case Down:
		return "down"
	default:
		return "stop"
	}
}
