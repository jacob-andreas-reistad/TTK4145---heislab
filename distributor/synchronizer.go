package distributor

import (
	"fmt"
	"heis/config"
	"heis/elevator"
	"heis/elevio"
	"heis/network/peers"
	"time"
)

type TempStorageType int

const (
	None TempStorageType = iota
	AddOrder
	RemoveOrder
	UpdateState
)

func Synchronizer(
	ElevID int,
	localStateCh <-chan elevator.State,
	peersCh <-chan peers.PeerUpdate,
	networkTx chan<- CommonState,
	networkRx <-chan CommonState,
	ackedCsCh chan<- CommonState,
	completedOrderCh <-chan elevio.ButtonEvent,
) {

	buttonEventCh := make(chan elevio.ButtonEvent, config.Buffer)
	go elevio.PollButtons(buttonEventCh)

	var cs CommonState
	var peers peers.PeerUpdate
	var newLocalState elevator.State
	var completedOrder elevio.ButtonEvent
	var newButtonEvent elevio.ButtonEvent
	var tempStorage TempStorageType

	heartbeat := time.NewTicker(config.HeartbeatTime)
	disconnectTimer := time.NewTimer(config.DisconnectTime)

	idle := true
	disconnected := false

	// Startup: ensure we reach a known floor
	//elevio.SetMotorDirection(elevio.MD_Down)

	//BIG ASS SWITCH CASE GOES HERE:
	for {
		select {
		//case heisen går offline
		// disconnected = true
		case <-disconnectTimer.C:
			cs.MakeOtherElevatorsUnavailable(ElevID)
			fmt.Println("Lost connction")
			disconnected = true

		//case heisen er ikke idle (se eksempel i EirikIsAChamp)
		//idle = false
		case peers = <-peersCh:
			cs.MakeOtherElevatorsUnavailable(ElevID)
			idle = false

		case <-heartbeat.C:
			networkTx <- cs

		default:
		}

		switch {
		case idle:
			select {
			case newButtonEvent = <-buttonEventCh: //new button press
				tempStorage = AddOrder
				cs.PrepNewCommonState(ElevID)
				cs.RegisterOrder(newButtonEvent, ElevID)
				cs.Acks[ElevID] = Confirmed
				idle = false

			case completedOrder = <-completedOrderCh: //order completed
				tempStorage = RemoveOrder
				cs.PrepNewCommonState(ElevID)
				cs.ClearOrder(completedOrder, ElevID)
				cs.Acks[ElevID] = Confirmed
				idle = false

			case newLocalState = <-localStateCh: //local state changes
				tempStorage = UpdateState
				cs.PrepNewCommonState(ElevID)
				cs.UpdateElevatorState(ElevID, newLocalState)
				cs.Acks[ElevID] = Confirmed
				idle = false

			case arrivedCs := <-networkRx: //new common state arrived while idle
				disconnectTimer = time.NewTimer(config.DisconnectTime)
				if arrivedCs.StateNum > cs.StateNum || (arrivedCs.Sender > cs.Sender && arrivedCs.StateNum == cs.StateNum) {
					cs = arrivedCs
					cs.MakeLostElevatorsUnavailable(peers)
					cs.Acks[ElevID] = Confirmed
					idle = false
				}
			default:
			}
		case disconnected:
			select {

			case newButtonEvent := <-buttonEventCh:
				if !cs.Elevators[ElevID].Current.MotorStop {
					cs.Acks[ElevID] = Confirmed
					cs.RegisterOrder(newButtonEvent, ElevID)
					ackedCsCh <- cs
				}

			case completedOrder := <-completedOrderCh:
				cs.Acks[ElevID] = Confirmed
				cs.ClearOrder(completedOrder, ElevID)
				ackedCsCh <- cs

			case newLocalState := <-localStateCh:
				if !(newLocalState.Obstructed || newLocalState.MotorStop) {
					cs.Acks[ElevID] = Confirmed
					cs.UpdateElevatorState(ElevID, newLocalState)
					ackedCsCh <- cs
				}

			case <-networkRx:
				if cs.Elevators[ElevID].CabCalls == [config.NumFloors]bool{} {
					fmt.Println("Connection restored to network.")
					disconnected = false
				} else {
					cs.Acks[ElevID] = Disconnected
					fmt.Println("Network connection lost. Cab calls will be cleared when completed.")
				}

			default:
			}

		case !idle:

		}
		_ = tempStorage
		//default (heisen er idle:)
		//switch
		//case idle:
		//select
		//case1, case2 osv.

		//case !idle:
		//select
		//case1,case2,case3 osv.

		//case offline:
		// select
		//case2,case2,case3 osv.

	}
}

// for {
// 	select {

// 	case btn := <-buttonCh:
// 		onButtonPress(btn, id, cs)

// 	case floor := <-floorCh:
// 		onFloorArrival(floor, id, cs)

// 	case <-doorTimeoutCh:
// 		onDoorTimeout(id, cs)

// 	case stop := <-stopCh:
// 		onStopButton(stop)

// 	case obstruction := <-obstructionCh:
// 		onObstruction(obstruction)
// 	}
// }

// func onButtonPress(btn elevio.ButtonEvent, id int, cs *CommonState) {

// 	// Register order
// 	if btn.Button == elevio.BT_Cab {
// 		cs.AddCabCall(btn.Floor, id)
// 	} else {
// 		cs.RegisterOrder(btn, id)
// 	}

// 	// SKRU PÅ LYS HER

// 	local := cs.Elevators[id]

// 	if local.Current == elevator.Idle {

// 		dir := motor_directions.ChooseDirection(local, cs)

// 		if dir != elevio.MD_Stop {
// 			elevio.SetMotorDirection(dir)
// 			cs.SetElevatorState(id, elevator.Moving)
// 		}
// 	}
// }

// func onFloorArrival(floor int, id int, cs *commonstate.SharedState) {

// 	elevio.SetFloorIndicator(floor)

// 	local := cs.Elevators[id]

// 	// Update floor inside elevator state
// 	local.Current.Floor = floor
// 	cs.Elevators[id] = local

// 	if motor_directions.ShouldStop(floor, local, cs) {

// 		elevio.SetMotorDirection(elevio.MD_Stop)

// 		elevator_doors.OpenDoor()

// 		cs.SetElevatorState(id, elevator.DoorOpen)

// 		clearOrdersAtFloor(floor, id, cs)

// 	} else {

// 		dir := motor_directions.ChooseDirection(local, cs)
// 		elevio.SetMotorDirection(dir)

// 		cs.SetElevatorState(id, elevator.Moving)
// 	}
// }

// func onDoorTimeout(id int, cs *commonstate.SharedState) {

// 	elevator_doors.CloseDoor()

// 	local := cs.Elevators[id]

// 	dir := motor_directions.ChooseDirection(local, cs)

// 	if dir == elevio.MD_Stop {

// 		cs.SetElevatorState(id, elevator.Idle)

// 	} else {

// 		elevio.SetMotorDirection(dir)
// 		cs.SetElevatorState(id, elevator.Moving)
// 	}
// }

// func onStopButton(stop bool) {

// 	if stop {

// 		elevio.SetMotorDirection(elevio.MD_Stop)
// 		elevio.SetStopLamp(true)

// 	} else {

// 		elevio.SetStopLamp(false)

// 	}
// }

// func onObstruction(obstruction bool) {

// 	if obstruction {

// 		elevator_doors.KeepDoorOpen()

// 	}
// }

// func clearOrdersAtFloor(floor int, id int, cs *commonstate.SharedState) {

// 	// Clear cab call
// 	cs.RemoveCabCall(floor, id)
// 	elevio.SetButtonLamp(elevio.BT_Cab, floor, false)

// 	// Clear hall calls
// 	cs.RemoveHallCall(floor, elevio.BT_HallUp)
// 	cs.RemoveHallCall(floor, elevio.BT_HallDown)

// 	elevio.SetButtonLamp(elevio.BT_HallUp, floor, false)
// 	elevio.SetButtonLamp(elevio.BT_HallDown, floor, false)
// }
