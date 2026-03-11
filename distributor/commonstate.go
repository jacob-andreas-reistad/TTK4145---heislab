package commonstate

import (
	"reflect"
	"root/config"
	"root/elevator"
	"root/elevio"
	"root/network/peers"
)

type AckState int

const (
	Pending AckState = iota
	Confirmed
	Offline
)

type Elevator struct {
	Current  elevator.State
	CabCalls [config.NumFloors]bool
}

type CommonState struct {
	Version int
	Sender  int

	Acks [config.NumElevators]AckState

	HallCalls [config.NumFloors][2]bool

	Elevators [config.NumElevators]Elevator
}

func (s *CommonState) RegisterOrder(btn elevio.ButtonEvent, id int) {
	switch btn.Button {
	case elevio.BT_Cab:
		s.Elevators[id].CabCalls[btn.Floor] = true
	default:
		s.HallCalls[btn.Floor][btn.Button] = true
	}
}

func (s *CommonState) ClearOrder(btn elevio.ButtonEvent, id int) {
	switch btn.Button {
	case elevio.BT_Cab:
		s.Elevators[id].CabCalls[btn.Floor] = false
	default:
		s.HallCalls[btn.Floor][btn.Button] = false
	}
}

func (s *CommonState) SetElevatorState(id int, state elevator.State) {
	info := s.Elevators[id]
	info.Current = state
	s.Elevators[id] = info
}

func (s *CommonState) AllAcknowledged(self int) bool {

	if s.Acks[self] == Offline {
		return false
	}

	for _, ack := range s.Acks {
		if ack == Pending {
			return false
		}
	}
	return true
}

func (s CommonState) SameState(other CommonState) bool {

	s.Acks = [config.NumElevators]AckState{}
	other.Acks = [config.NumElevators]AckState{}

	return reflect.DeepEqual(s, other)
}

func (s *CommonState) HandlePeerUpdate(update peers.PeerUpdate) {

	for _, lost := range update.Lost {
		s.Acks[lost] = Offline
	}
}

func (s *CommonState) InitializeSolo(id int) {

	for i := range s.Acks {
		if i != id {
			s.Acks[i] = Offline
		}
	}
}

func (s *CommonState) BeginUpdate(id int) {

	s.Version++
	s.Sender = id

	for i := range s.Acks {
		if s.Acks[i] == Confirmed {
			s.Acks[i] = Pending
		}
	}
}
