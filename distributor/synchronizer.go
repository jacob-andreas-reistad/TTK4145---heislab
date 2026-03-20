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

// Coordinates elevator state over the network, handling button presses, completed orders, and peer connections.
func Synchronizer(
	elevID int,
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
	var peerUpdate peers.PeerUpdate
	var newLocalState elevator.State
	var completedOrder elevio.ButtonEvent
	var newButtonEvent elevio.ButtonEvent
	var tempStorage TempStorageType
	offlineHallCalls := [config.NumFloors][2]bool{}

	heartbeat := time.NewTicker(config.HeartbeatTime)
	disconnectTimer := time.NewTimer(config.DisconnectTime)

	idle := true
	disconnected := false

	// Startup: ensure we reach a known floor
	for {
		select {

		case <-disconnectTimer.C:
			cs.MakeOtherElevatorsUnavailable(elevID)
			fmt.Printf("[network] Elevator %d disconnected - running solo\n", elevID)
			disconnected = true

		case peerUpdate = <-peersCh:
			cs.MakeLostElevatorsUnavailable(peerUpdate)
			for _, lostID := range peerUpdate.Lost {
				fmt.Printf("[network] Elevator %s lost connection\n", lostID)
			}
			if peerUpdate.New != "" {
				fmt.Printf("[network] Elevator %s joined the network\n", peerUpdate.New)
			}
			idle = false

		case <-heartbeat.C:
			networkTx <- cs

		default:
		}

		switch {
		case disconnected:
			select {

			case arrivedCs := <-networkRx:
				fmt.Println("[network] Connection restored - merging offline state")

				// Merge hall calls created while offline into the network state
				for f := range offlineHallCalls {
					for b := range offlineHallCalls[f] {
						if offlineHallCalls[f][b] {
							arrivedCs.HallCalls[f][b] = true
						}
					}
				}

				// Merge local cab calls (only we know our cab state while offline)
				for f := 0; f < config.NumFloors; f++ {
					if cs.Elevators[elevID].CabCalls[f] {
						arrivedCs.Elevators[elevID].CabCalls[f] = true
					}
				}

				// Update our elevator state in the network view
				arrivedCs.Elevators[elevID].Current = cs.Elevators[elevID].Current

				// Clear offline tracking
				offlineHallCalls = [config.NumFloors][2]bool{}

				cs = arrivedCs
				cs.PrepNewCommonState(elevID)
				cs.MakeLostElevatorsUnavailable(peerUpdate)
				cs.Acks[elevID] = Confirmed
				disconnectTimer = time.NewTimer(config.DisconnectTime)
				disconnected = false
				idle = false

			// Handle new events while disconnected
			case newButtonEvent := <-buttonEventCh:
				if newButtonEvent.Button != elevio.BT_Cab || !cs.Elevators[elevID].Current.MotorStop {
					cs.Acks[elevID] = Confirmed
					cs.RegisterOrder(newButtonEvent, elevID)
					if newButtonEvent.Button != elevio.BT_Cab {
						offlineHallCalls[newButtonEvent.Floor][newButtonEvent.Button] = true
					}
					ackedCsCh <- cs
				}

			// Handle completed orders while disconnected
			case completedOrder := <-completedOrderCh:
				cs.Acks[elevID] = Confirmed
				cs.ClearOrder(completedOrder, elevID)
				if completedOrder.Button != elevio.BT_Cab {
					offlineHallCalls[completedOrder.Floor][completedOrder.Button] = false
				}
				ackedCsCh <- cs

			// Handle local state changes while disconnected
			case newLocalState := <-localStateCh:
				if !(newLocalState.Obstructed || newLocalState.MotorStop) {
					cs.Acks[elevID] = Confirmed
					cs.UpdateElevatorState(elevID, newLocalState)
					ackedCsCh <- cs
				}

			default:
			}

		case idle:
			select {

			// New button press
			case newButtonEvent = <-buttonEventCh:
				tempStorage = AddOrder
				cs.PrepNewCommonState(elevID)
				cs.RegisterOrder(newButtonEvent, elevID)
				cs.Acks[elevID] = Confirmed
				idle = false

			// Order completed
			case completedOrder = <-completedOrderCh:
				tempStorage = RemoveOrder
				cs.PrepNewCommonState(elevID)
				cs.ClearOrder(completedOrder, elevID)
				cs.Acks[elevID] = Confirmed
				idle = false

			// Local state changes
			case newLocalState = <-localStateCh:
				tempStorage = UpdateState
				cs.PrepNewCommonState(elevID)
				cs.UpdateElevatorState(elevID, newLocalState)
				cs.Acks[elevID] = Confirmed
				idle = false

			// New common state arrived while idle
			case arrivedCs := <-networkRx:
				disconnectTimer = time.NewTimer(config.DisconnectTime)
				if arrivedCs.StateNum > cs.StateNum || (arrivedCs.Sender > cs.Sender && arrivedCs.StateNum == cs.StateNum) {
					cs = arrivedCs
					cs.MakeLostElevatorsUnavailable(peerUpdate)
					cs.Acks[elevID] = Confirmed
					idle = false
				}
			default:
			}

		// Handle events while not idle, but prioritize network updates to ensure we stay in sync
		case !idle:
			select {

			// Handle new button presses while not idle
			case completedOrder = <-completedOrderCh:
				cs.ClearOrder(completedOrder, elevID)
				ackedCsCh <- cs

			// New common state arrived while not idle
			case arrivedCs := <-networkRx:
				if arrivedCs.StateNum < cs.StateNum {
					break
				}

				disconnectTimer = time.NewTimer(config.DisconnectTime)

				switch {

				// If the arrived state is newer
				case arrivedCs.StateNum > cs.StateNum || (arrivedCs.Sender > cs.Sender && arrivedCs.StateNum == cs.StateNum):
					cs = arrivedCs
					cs.Acks[elevID] = Confirmed
					cs.MakeLostElevatorsUnavailable(peerUpdate)

				// If the arrived state is the same but we haven't acknowledged it yet
				case arrivedCs.AllAcknowledged(elevID):
					cs = arrivedCs
					cs.MakeLostElevatorsUnavailable(peerUpdate)
					cs.printOrders()
					ackedCsCh <- cs

					switch {

					// If we are the sender of the state
					case cs.Sender != elevID && tempStorage != None:
						cs.PrepNewCommonState(elevID)

						switch tempStorage {

						// If we have a new order, we need to register it in the network state
						case AddOrder:
							cs.RegisterOrder(newButtonEvent, elevID)
							cs.Acks[elevID] = Confirmed

						// If we completed an order, we need to clear it from the network state
						case RemoveOrder:
							cs.ClearOrder(completedOrder, elevID)
							cs.Acks[elevID] = Confirmed

						// If we had a local state change, we need to update the network state
						case UpdateState:
							cs.UpdateElevatorState(elevID, newLocalState)
							cs.Acks[elevID] = Confirmed
						}

					// If we are not the sender, but we had a pending change that is now acknowledged
					case cs.Sender == elevID && tempStorage != None:
						tempStorage = None
						idle = true

					// If we are not the sender and we didn't have a pending change
					default:
						idle = true
					}

				// If the arrived state is the same but we haven't acknowledged it yet
				case cs.CheckSameState(arrivedCs):
					cs = arrivedCs
					cs.Acks[elevID] = Confirmed
					cs.MakeLostElevatorsUnavailable(peerUpdate)
				default:
				}
			default:
			}
		}
	}
}
