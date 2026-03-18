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
			cs.MakeLostElevatorsUnavailable(peers)
			idle = false

		case <-heartbeat.C:
			networkTx <- cs

		default:
		}

		switch {
		case disconnected:
			select {

			case <-networkRx:
				if cs.Elevators[ElevID].CabCalls == [config.NumFloors]bool{} {
					fmt.Println("Connection restored to network.")
					disconnected = false
				} else {
					cs.Acks[ElevID] = Disconnected
					fmt.Println("Network connection lost. Cab calls will be cleared when completed.")
				}

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

			default:
			}

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
		case !idle:
			select {
			case arrivedCs := <-networkRx: //new common state arrived while not idle
				if arrivedCs.StateNum < cs.StateNum {
					break
				}

				disconnectTimer = time.NewTimer(config.DisconnectTime)

				switch {
				case arrivedCs.StateNum > cs.StateNum || (arrivedCs.Sender > cs.Sender && arrivedCs.StateNum == cs.StateNum):
					cs = arrivedCs
					cs.Acks[ElevID] = Confirmed
					cs.MakeLostElevatorsUnavailable(peers)

				case arrivedCs.AllAcknowledged(ElevID):
					cs = arrivedCs
					ackedCsCh <- cs

					switch {
					case cs.Sender != ElevID && tempStorage != None:
						cs.PrepNewCommonState(ElevID)

						switch tempStorage {
						case AddOrder:
							cs.RegisterOrder(newButtonEvent, ElevID)
							cs.Acks[ElevID] = Confirmed

						case RemoveOrder:
							cs.ClearOrder(completedOrder, ElevID)
							cs.Acks[ElevID] = Confirmed

						case UpdateState:
							cs.UpdateElevatorState(ElevID, newLocalState)
							cs.Acks[ElevID] = Confirmed
						}
					case cs.Sender == ElevID && tempStorage != None:
						tempStorage = None
						idle = true

					default:
						idle = true
					}

				case cs.CheckSameState(arrivedCs):
					cs = arrivedCs
					cs.Acks[ElevID] = Confirmed
					cs.MakeLostElevatorsUnavailable(peers)

				default:
				}
			default:
			}
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
