package lights

import (
	"heis/config"
	"heis/distributor"
	"heis/elevio"
)

func SetPanelLights(CommonState distributor.CommonState, ElevatorNumber int) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for buttonType := 0; buttonType < 2; buttonType++ {
			if CommonState.HallCalls[floor][buttonType] {
				elevio.SetButtonLamp(elevio.ButtonType(buttonType), floor, true)
			} else {
				elevio.SetButtonLamp(elevio.ButtonType(buttonType), floor, false)
			}
		}
	}

	for floor := 0; floor < config.NumFloors; floor++ {
		if CommonState.Elevators[ElevatorNumber].CabCalls[floor] {
			elevio.SetButtonLamp(elevio.BT_Cab, floor, true)
		} else {
			elevio.SetButtonLamp(elevio.BT_Cab, floor, false)
		}
	}
}
