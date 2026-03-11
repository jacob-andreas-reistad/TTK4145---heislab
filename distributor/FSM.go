package distributor

import (
	"heis/elevator"
	"heis/elevio"
)

func FSM(
	id int,
	cs *CommonState,

	buttonCh <-chan elevio.ButtonEvent,
	floorCh <-chan int,
	doorTimeoutCh <-chan bool,
	stopCh <-chan bool,
	obstructionCh <-chan bool,
) {

	// Startup: ensure we reach a known floor
	elevio.SetMotorDirection(elevio.MD_Down)

	for {
		select {

		case btn := <-buttonCh:
			onButtonPress(btn, id, cs)

		case floor := <-floorCh:
			onFloorArrival(floor, id, cs)

		case <-doorTimeoutCh:
			onDoorTimeout(id, cs)

		case stop := <-stopCh:
			onStopButton(stop)

		case obstruction := <-obstructionCh:
			onObstruction(obstruction)
		}
	}
}

func onButtonPress(btn elevio.ButtonEvent, id int, cs *CommonState) {

	// Register order
	if btn.Button == elevio.BT_Cab {
		cs.AddCabCall(btn.Floor, id)
	} else {
		cs.RegisterOrder(btn, id)
	}

	// SKRU PÅ LYS HER

	local := cs.Elevators[id]

	if local.Current == elevator.Idle {

		dir := motor_directions.ChooseDirection(local, cs)

		if dir != elevio.MD_Stop {
			elevio.SetMotorDirection(dir)
			cs.SetElevatorState(id, elevator.Moving)
		}
	}
}

func onFloorArrival(floor int, id int, cs *commonstate.SharedState) {

	elevio.SetFloorIndicator(floor)

	local := cs.Elevators[id]

	// Update floor inside elevator state
	local.Current.Floor = floor
	cs.Elevators[id] = local

	if motor_directions.ShouldStop(floor, local, cs) {

		elevio.SetMotorDirection(elevio.MD_Stop)

		elevator_doors.OpenDoor()

		cs.SetElevatorState(id, elevator.DoorOpen)

		clearOrdersAtFloor(floor, id, cs)

	} else {

		dir := motor_directions.ChooseDirection(local, cs)
		elevio.SetMotorDirection(dir)

		cs.SetElevatorState(id, elevator.Moving)
	}
}

func onDoorTimeout(id int, cs *commonstate.SharedState) {

	elevator_doors.CloseDoor()

	local := cs.Elevators[id]

	dir := motor_directions.ChooseDirection(local, cs)

	if dir == elevio.MD_Stop {

		cs.SetElevatorState(id, elevator.Idle)

	} else {

		elevio.SetMotorDirection(dir)
		cs.SetElevatorState(id, elevator.Moving)
	}
}

func onStopButton(stop bool) {

	if stop {

		elevio.SetMotorDirection(elevio.MD_Stop)
		elevio.SetStopLamp(true)

	} else {

		elevio.SetStopLamp(false)

	}
}

func onObstruction(obstruction bool) {

	if obstruction {

		elevator_doors.KeepDoorOpen()

	}
}

func clearOrdersAtFloor(floor int, id int, cs *commonstate.SharedState) {

	// Clear cab call
	cs.RemoveCabCall(floor, id)
	elevio.SetButtonLamp(elevio.BT_Cab, floor, false)

	// Clear hall calls
	cs.RemoveHallCall(floor, elevio.BT_HallUp)
	cs.RemoveHallCall(floor, elevio.BT_HallDown)

	elevio.SetButtonLamp(elevio.BT_HallUp, floor, false)
	elevio.SetButtonLamp(elevio.BT_HallDown, floor, false)
}
