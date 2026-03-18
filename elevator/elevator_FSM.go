// This file contains the FSM for the elevator system... tbc

package elevator

import (
	"fmt"
	"heis/config"
	"heis/elevio"
	"time"
)

type Behaviour int

type State struct {
	Floor           int
	Direction       MotorDirection
	Obstructed      bool
	Behaviour       Behaviour
	MotorStop       bool
	LastServedFloor int
}

const (
	Idle Behaviour = iota
	Moving
	DoorsOpen
)

func (bh Behaviour) ToString() string {
	switch bh {
	case Idle:
		return "idle"
	case Moving:
		return "moving"
	case DoorsOpen:
		return "doorOpen"
	default:
		panic("Invalid behaviour")
	}
}

func Elevator(newOrderCh <-chan Order, orderDoneCh chan<- elevio.ButtonEvent, stateUpdateCh chan<- State, savedDir MotorDirection) {
	openDoorCh := make(chan bool)
	closeDoorCh := make(chan bool)
	doorObstructedCh := make(chan bool)
	floorEnteredCh := make(chan int)
	motorCh := make(chan bool, 1)

	go doors(closeDoorCh, openDoorCh, doorObstructedCh)
	go elevio.PollFloorSensor(floorEnteredCh)

	elevio.SetMotorDirection(elevio.MD_Stop)
	time.Sleep(50 * time.Millisecond)
	initialFloor := elevio.GetFloor()
	var state State
	if initialFloor != -1 {
		state = State{Direction: savedDir, Behaviour: Idle, Floor: initialFloor, LastServedFloor: -1}
		elevio.SetFloorIndicator(initialFloor)
		stateUpdateCh <- state
	} else {
		elevio.SetMotorDirection(savedDir.motor_direction())
		state = State{Direction: savedDir, Behaviour: Moving, LastServedFloor: -1}
	}

	var orders Order

	motorTimer := time.NewTimer(config.WatchdogTime)
	obstructionTimer := time.NewTimer(config.WatchdogTime)
	motorTimer.Stop()
	obstructionTimer.Stop()

	for {
		select {

		case <-closeDoorCh:
			switch state.Behaviour {
			case DoorsOpen:
				switch {
				case orders.has_orders(state.Direction, state.Floor):
					elevio.SetMotorDirection(state.Direction.motor_direction())
					state.Behaviour = Moving
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorCh <- false
					stateUpdateCh <- state

				case orders.has_orders(state.Direction.opposite(), state.Floor):
					state.Direction = state.Direction.opposite()
					elevio.SetMotorDirection(state.Direction.motor_direction())
					state.Behaviour = Moving
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorCh <- false
					stateUpdateCh <- state

				default:
					state.Behaviour = Idle
					stateUpdateCh <- state
				}
			default:
				panic(fmt.Sprintf("Received close door signal while not in DoorsOpen state. Current state: %s", state.Behaviour.ToString()))
			}
		case state.Floor = <-floorEnteredCh:
			elevio.SetFloorIndicator(state.Floor)
			motorTimer.Stop()
			motorCh <- false

			switch state.Behaviour {
			case Moving:
				switch {
				case orders[state.Floor][state.Direction.button_type()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.LastServedFloor = state.Floor
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders[state.Floor][elevio.BT_Cab] && orders.has_orders(state.Direction, state.Floor):
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders[state.Floor][elevio.BT_Cab] && !orders[state.Floor][state.Direction.opposite().button_type()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders.has_orders(state.Direction.opposite(), state.Floor) &&
					!(state.Floor == 0 && state.Direction == Down) &&
					!(state.Floor == config.NumFloors-1 && state.Direction == Up):
					motorTimer = time.NewTimer(config.WatchdogTime)

				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
					stateUpdateCh <- state
				}
			default:
				// Idle or DoorsOpen: just update floor position
			}

		case orders = <-newOrderCh:
			switch state.Behaviour {
			case Idle:
				switch {
				case orders[state.Floor][state.Direction.button_type()] || orders[state.Floor][elevio.BT_Cab]:
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders[state.Floor][state.Direction.opposite().button_type()] && state.Floor != state.LastServedFloor:
					openDoorCh <- true
					state.Direction = state.Direction.opposite()
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders.has_orders(state.Direction, state.Floor):
					elevio.SetMotorDirection(state.Direction.motor_direction())
					state.Behaviour = Moving
					stateUpdateCh <- state
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorCh <- false

				case orders.has_orders(state.Direction.opposite(), state.Floor):
					state.Direction = state.Direction.opposite()
					elevio.SetMotorDirection(state.Direction.motor_direction())
					state.Behaviour = Moving
					stateUpdateCh <- state
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorCh <- false

				default:
				}

			case DoorsOpen:
				switch {
				case orders[state.Floor][elevio.BT_Cab] || orders[state.Floor][state.Direction.button_type()]:
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
				}

			case Moving:

			default:
				panic("Orders in wrong state")
			}
		case <-motorTimer.C:
			if !state.MotorStop {
				fmt.Println("motor power lost")
				state.MotorStop = true
				stateUpdateCh <- state
			}

		case motor := <-motorCh:
			if state.MotorStop {
				fmt.Println("motor power restored")
				state.MotorStop = motor
				stateUpdateCh <- state
			}

		case obstructed := <-doorObstructedCh:
			if obstructed != state.Obstructed {
				state.Obstructed = obstructed
				if obstructed {
					obstructionTimer = time.NewTimer(config.WatchdogTime)
				} else {
					obstructionTimer.Stop()
				}
				stateUpdateCh <- state
			}
		case <-obstructionTimer.C:
			if !state.MotorStop {
				fmt.Println("obstruction timeout - marking unavailable")
				state.MotorStop = true
				stateUpdateCh <- state
			}
		}
	}
}
