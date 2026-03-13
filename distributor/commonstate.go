package distributor

import (
	"heis/config"
	"heis/elevator"
	"heis/elevio"
	"heis/network/peers"
	"reflect"
	"strconv"
)

type AckState int

const (
	Pending AckState = iota
	Confirmed
	Disconnected
)

type Elevator struct {
	Current  elevator.State
	CabCalls [config.NumFloors]bool
}

type CommonState struct {
	StateNum int
	Sender   int

	Acks [config.NumElevators]AckState

	HallCalls [config.NumFloors][2]bool

	Elevators [config.NumElevators]Elevator
}

// Bør evt splittes i to, en hallcall og en cabcall
func (cs *CommonState) RegisterOrder(btn elevio.ButtonEvent, id int) {
	switch btn.Button {
	case elevio.BT_Cab:
		cs.Elevators[id].CabCalls[btn.Floor] = true
	default:
		cs.HallCalls[btn.Floor][btn.Button] = true
	}
}

func (cs *CommonState) ClearOrder(btn elevio.ButtonEvent, id int) {
	switch btn.Button {
	case elevio.BT_Cab:
		cs.Elevators[id].CabCalls[btn.Floor] = false
	default:
		cs.HallCalls[btn.Floor][btn.Button] = false
	}
}

func (cs *CommonState) UpdateElevatorState(id int, state elevator.State) {
	info := cs.Elevators[id]
	info.Current = state
	cs.Elevators[id] = info
}

func (cs *CommonState) AllAcknowledged(self int) bool {

	if cs.Acks[self] == Disconnected {
		return false
	}

	for _, ack := range cs.Acks {
		if ack == Pending {
			return false
		}
	}
	return true
}

func (cs CommonState) CheckSameState(newCs CommonState) bool {

	cs.Acks = [config.NumElevators]AckState{}
	newCs.Acks = [config.NumElevators]AckState{}

	return reflect.DeepEqual(cs, newCs)
}

func (cs *CommonState) MakeLostElevatorsUnavailable(peerList peers.PeerUpdate) {

	for _, lost := range peerList.Lost {
		lostID, err := strconv.Atoi(lost)
		if err != nil {
			continue
		}
		cs.Acks[lostID] = Disconnected
	}

}

func (cs *CommonState) MakeOtherElevatorsUnavailable(id int) {

	for elevator := range cs.Acks {
		if elevator != id {
			cs.Acks[elevator] = Disconnected
		}
	}
}

func (cs *CommonState) PrepNewCommonState(id int) {

	cs.StateNum++
	cs.Sender = id

	for i := range cs.Acks {
		if cs.Acks[i] == Confirmed {
			cs.Acks[i] = Pending
		}
	}
}
