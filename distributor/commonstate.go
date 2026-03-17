package distributor

import (
	"fmt"
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

func (cs *CommonState) printOrders() {
	fmt.Printf("  %-10s", "")
	for f := 0; f < config.NumFloors; f++ {
		fmt.Printf(" F%-3d", f)
	}
	fmt.Println()

	rows := []struct {
		label  string
		values func(f int) bool
	}{
		{"Hall Up", func(f int) bool { return cs.HallCalls[f][elevio.BT_HallUp] }},
		{"Hall Dn", func(f int) bool { return cs.HallCalls[f][elevio.BT_HallDown] }},
	}
	for id := 0; id < config.NumElevators; id++ {
		idCopy := id
		rows = append(rows, struct {
			label  string
			values func(f int) bool
		}{fmt.Sprintf("Cab [%d]", idCopy), func(f int) bool { return cs.Elevators[idCopy].CabCalls[f] }})
	}

	for _, row := range rows {
		fmt.Printf("  %-10s", row.label)
		for f := 0; f < config.NumFloors; f++ {
			if row.values(f) {
				fmt.Printf("  *  ")
			} else {
				fmt.Printf("  .  ")
			}
		}
		fmt.Println()
	}
}

// Bør evt splittes i to, en hallcall og en cabcall
func (cs *CommonState) RegisterOrder(btn elevio.ButtonEvent, id int) {
	switch btn.Button {
	case elevio.BT_Cab:
		cs.Elevators[id].CabCalls[btn.Floor] = true
	default:
		cs.HallCalls[btn.Floor][btn.Button] = true
	}
	btnName := map[elevio.ButtonType]string{elevio.BT_HallUp: "HallUp", elevio.BT_HallDown: "HallDown", elevio.BT_Cab: "Cab"}
	fmt.Printf("[NEW ORDER] floor=%d  button=%s\n", btn.Floor, btnName[btn.Button])
	cs.printOrders()
}

func (cs *CommonState) ClearOrder(btn elevio.ButtonEvent, id int) {
	switch btn.Button {
	case elevio.BT_Cab:
		cs.Elevators[id].CabCalls[btn.Floor] = false
	default:
		cs.HallCalls[btn.Floor][btn.Button] = false
	}
	btnName := map[elevio.ButtonType]string{elevio.BT_HallUp: "HallUp", elevio.BT_HallDown: "HallDown", elevio.BT_Cab: "Cab"}
	fmt.Printf("[ORDER DONE] floor=%d  button=%s\n", btn.Floor, btnName[btn.Button])
	cs.printOrders()
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
	for i := range cs.Acks {
		found := false
		for _, peer := range peerList.Peers {
			peerID, err := strconv.Atoi(peer)
			if err != nil {
				continue
			}
			if peerID == i {
				found = true
				break
			}
		}
		if !found {
			cs.Acks[i] = Disconnected
		}
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
