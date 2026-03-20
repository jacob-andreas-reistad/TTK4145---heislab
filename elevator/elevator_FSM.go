package elevator

import (
	"fmt"
	"heis/config"
	"heis/elevio"
	"time"
)

// State represents the current state of the elevator, including its floor, direction, whether it is obstructed,
// its behaviour (idle, moving or doors open), whether the motor is stopped and the last served floor
type State struct {
	Floor           int
	Direction       MotorDirection
	Obstructed      bool
	Behaviour       Behaviour
	MotorStop       bool
	LastServedFloor int
}

// Behaviour represents the current behaviour of the elevator, which can be Idle, Moving or DoorsOpen
type Behaviour int

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
		panic("invalid behaviour")
	}
}

// Elevator is the main function for controlling the elevator. It receives new orders, updates the state of the elevator,
// and handles the logic for moving, opening and closing doors, and dealing with obstructions and motor failures.
func Elevator(newOrderCh <-chan Order, orderDoneCh chan<- elevio.ButtonEvent, stateUpdateCh chan<- State, savedDir MotorDirection) {
	openDoorCh := make(chan bool)
	closeDoorCh := make(chan bool)
	doorObstructedCh := make(chan bool)
	floorEnteredCh := make(chan int)
	motorCh := make(chan bool, 1)

	go doors(closeDoorCh, openDoorCh, doorObstructedCh)
	go elevio.PollFloorSensor(floorEnteredCh)

	elevio.SetMotorDirection(elevio.MD_Stop)
	time.Sleep(config.InitSettleTime)
	initialFloor := elevio.GetFloor()

	var state State

	if initialFloor != -1 {
		state = State{Direction: savedDir, Behaviour: Idle, Floor: initialFloor, LastServedFloor: -1}
		elevio.SetFloorIndicator(initialFloor)
		stateUpdateCh <- state
	} else {
		elevio.SetMotorDirection(savedDir.motorDirection())
		state = State{Direction: savedDir, Behaviour: Moving, LastServedFloor: -1}
	}

	motorTimer := time.NewTimer(config.WatchdogTime)
	obstructionTimer := time.NewTimer(config.WatchdogTime)
	motorRetryTimer := time.NewTimer(config.MotorRetryTime)
	motorTimer.Stop()
	obstructionTimer.Stop()
	motorRetryTimer.Stop()

	var orders Order

	for {
		select {

		// Handle close door signal: if the doors are open, check for orders and decide whether to continue moving, switch direction or go idle
		case <-closeDoorCh:
			switch state.Behaviour {
			case DoorsOpen:
				switch {

				// If there are orders in the current direction
				case orders.hasOrders(state.Direction, state.Floor):
					elevio.SetMotorDirection(state.Direction.motorDirection())
					state.Behaviour = Moving
					motorTimer = time.NewTimer(config.WatchdogTime)
					select {
					case motorCh <- false:
					default:
					}
					stateUpdateCh <- state

				// If there are orders in the opposite direction
				case orders.hasOrders(state.Direction.opposite(), state.Floor):
					state.Direction = state.Direction.opposite()
					elevio.SetMotorDirection(state.Direction.motorDirection())
					state.Behaviour = Moving
					motorTimer = time.NewTimer(config.WatchdogTime)
					select {
					case motorCh <- false:
					default:
					}
					stateUpdateCh <- state

				// If there are no orders
				default:
					state.Behaviour = Idle
					stateUpdateCh <- state
				}
			default:
				panic(fmt.Sprintf("Received close door signal while not in DoorsOpen state. Current state: %s", state.Behaviour.ToString()))
			}

		// Handle floor entered signal: update the floor indicator, check for orders and decide whether to open doors, continue moving or go idle
		case state.Floor = <-floorEnteredCh:
			elevio.SetFloorIndicator(state.Floor)
			motorTimer.Stop()
			select {
			case motorCh <- false:
			default:
			}

			switch state.Behaviour {

			// If we are moving, check for orders and decide whether to stop and open doors, switch direction or continue moving
			case Moving:
				switch {

				// If there are orders for the current floor and direction
				case orders[state.Floor][state.Direction.buttonType()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					orderComplete(orders, state.Direction, state.Floor, orderDoneCh)
					state.LastServedFloor = state.Floor
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				// If there are orders for the current floor in the opposite direction, stop and open doors,
				// but only if we haven't just served this floor in the current direction
				case orders[state.Floor][elevio.BT_Cab] && orders.hasOrders(state.Direction, state.Floor):
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					orderComplete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				// If there are orders for the current floor in the opposite direction, but we just served this floor in the current direction
				case orders[state.Floor][elevio.BT_Cab] && !orders[state.Floor][state.Direction.opposite().buttonType()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorCh <- true
					orderComplete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				// If there are orders in the opposite direction, but not for the current floor
				case orders.hasOrders(state.Direction.opposite(), state.Floor) &&
					!(state.Floor == 0 && state.Direction == Down) &&
					!(state.Floor == config.NumFloors-1 && state.Direction == Up):
					motorTimer = time.NewTimer(config.WatchdogTime)

				// If there are no orders in either direction
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
					stateUpdateCh <- state
				}
			default:
				// Idle or DoorsOpen: just update floor position
			}

		// Handle new orders: depending on the current behaviour and orders, decide whether to open doors, start moving or switch direction
		case orders = <-newOrderCh:
			switch state.Behaviour {

			// If we are idle, check for orders and decide whether to open doors, start moving or switch direction
			case Idle:
				switch {

				// If there are orders for the current floor and direction
				case orders[state.Floor][state.Direction.buttonType()] || orders[state.Floor][elevio.BT_Cab]:
					openDoorCh <- true
					orderComplete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				// If there is an order at the current floor for the opposite direction, and this floor wasn't the last served floor
				case orders[state.Floor][state.Direction.opposite().buttonType()] && state.Floor != state.LastServedFloor:
					openDoorCh <- true
					state.Direction = state.Direction.opposite()
					orderComplete(orders, state.Direction, state.Floor, orderDoneCh)
					state.Behaviour = DoorsOpen
					stateUpdateCh <- state

				// If there are orders above/below the current floor in the current direction
				case orders.hasOrders(state.Direction, state.Floor):
					elevio.SetMotorDirection(state.Direction.motorDirection())
					state.Behaviour = Moving
					stateUpdateCh <- state
					motorTimer = time.NewTimer(config.WatchdogTime)
					select {
					case motorCh <- false:
					default:
					}

				// If there are orders in the opposite direction
				case orders.hasOrders(state.Direction.opposite(), state.Floor):
					state.Direction = state.Direction.opposite()
					elevio.SetMotorDirection(state.Direction.motorDirection())
					state.Behaviour = Moving
					stateUpdateCh <- state
					motorTimer = time.NewTimer(config.WatchdogTime)
					select {
					case motorCh <- false:

					// If the motor channel is full, it means we are already handling a motor failure, so we don't need to send another signal
					default:
					}

				// If no orders, stay idle
				default:
				}

			// If we have new orders while the doors are open, check if we need to stay and serve more orders at this floor,
			// or if we can close the doors and start moving
			case DoorsOpen:
				switch {

				// If there are orders for the current floor and direction, or if there are cab orders for the current floor
				case orders[state.Floor][elevio.BT_Cab] || orders[state.Floor][state.Direction.buttonType()]:
					openDoorCh <- true
					orderComplete(orders, state.Direction, state.Floor, orderDoneCh)
				}

			// If we get new orders while moving, we don't need to do anything,
			// as the logic for handling new orders will be triggered when we reach the next floor
			case Moving:

			// If we receive new orders while in an unknown state
			default:
				panic("Orders in wrong state")
			}

		// Handle motor failure:
		case <-motorTimer.C:
			if !state.MotorStop {
				fmt.Println("motor power lost")
				state.MotorStop = true
				stateUpdateCh <- state
				motorRetryTimer = time.NewTimer(500 * time.Millisecond)
			}

		// Handle motor power restoration:
		case <-motorRetryTimer.C:
			if state.MotorStop {
				elevio.SetMotorDirection(state.Direction.motorDirection())
				motorRetryTimer = time.NewTimer(500 * time.Millisecond)
			}

		// If we receive a signal on the motor channel, it means we have either just detected a motor failure or just restored power
		case motor := <-motorCh:
			if state.MotorStop {
				fmt.Println("motor power restored")
				state.MotorStop = motor
				motorRetryTimer.Stop()
				stateUpdateCh <- state
			}

		// Handle door obstruction signals:
		case obstructed := <-doorObstructedCh:
			if obstructed != state.Obstructed {
				state.Obstructed = obstructed
				if obstructed {
					obstructionTimer = time.NewTimer(config.WatchdogTime)
				} else {
					obstructionTimer.Stop()
					state.MotorStop = false
					stateUpdateCh <- state
				}
			}

		// Handle door obstruction timeout:
		case <-obstructionTimer.C:
			if !state.MotorStop {
				fmt.Println("obstruction timeout - marking unavailable")
				state.MotorStop = true
				stateUpdateCh <- state
			}
		}
	}
}
