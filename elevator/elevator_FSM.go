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
	Floor      int
	Direction  MotorDirection
	Obstructed bool
	Behaviour  Behaviour
	MotorStop  bool
}

const (
	Idle Behaviour = iota
	Moving
	DoorsOpen
)

func (bh Behaviour) to_string() string {
	switch bh {
	case Idle:
		return "Idle"
	case Moving:
		return "Moving"
	case DoorsOpen:
		return "Doors Open"
	default:
		panic("Invalid behaviour")
	}
}

func Elevator(newOrderCh <-chan Order, orderDoneCh chan<- elevio.ButtonEvent, stateUpdateCh chan<- State) {
	openDoorCh := make(chan bool)
	closeDoorCh := make(chan bool)
	doorObstructedCh := make(chan bool)
	floorEnteredCh := make(chan int)
	motorCh := make(chan bool)

	go doors(closeDoorCh, openDoorCh, doorObstructedCh)
	go elevio.PollFloorSensor(floorEnteredCh)

	elevio.SetMotorDirection(elevio.MD_Down)
	state := State{Direction: Down, Behaviour: Moving}

	var orders Order

	motorTimer := time.NewTimer(config.WatchdogTime)
	motorTimer.Stop()

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

				case orders[state.Floor][state.Direction.opposite()]:
					openDoorCh <- true
					state.Direction = state.Direction.opposite()
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
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
				panic(fmt.Sprintf("Received close door signal while not in DoorsOpen state. Current state: %s", state.Behaviour.to_string()))
			}
		case state.Floor = <-floorEnteredCh:
			elevio.SetFloorIndicator(state.Floor)
			motorTimer.Stop()
			motorCh <- false

			switch state.Behaviour {
			case Moving:
				switch {
				case orders[state.Floor][state.Direction]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders[state.Floor][elevio.BT_Cab] && orders.has_orders(state.Direction, state.Floor):
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders[state.Floor][elevio.BT_Cab] && !orders[state.Floor][state.Direction.opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders.has_orders(state.Direction.opposite(), state.Floor):
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorCh <- false

				case orders[state.Floor][state.Direction.opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					state.Direction = state.Direction.opposite()
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders.has_orders(state.Direction, state.Floor):
					state.Direction = state.Direction.opposite()
					elevio.SetMotorDirection(state.Direction.motor_direction())
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorCh <- false
					stateUpdateCh <- state

				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
					stateUpdateCh <- state
				}
			default:
				panic("Recieved floor entered in wrong state")
			}
			stateUpdateCh <- state

		case orders = <-newOrderCh:
			switch state.Behaviour {
			case Idle:
				switch {
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

				case orders[state.Floor][state.Direction] || orders[state.Floor][elevio.BT_Cab]:
					openDoorCh <- true
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				case orders[state.Floor][state.Direction.opposite()]:
					openDoorCh <- true
					state.Direction = state.Direction.opposite()
					order_complete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				default:
				}

			case DoorsOpen:
				switch {
				case orders[state.Floor][elevio.BT_Cab] || orders[state.Floor][state.Direction]:
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
				stateUpdateCh <- state
			}
		}
	}
}
