package elevator

import (
	"heis/config"
	"heis/elevio"
	"time"
)

type DoorState int

const (
	Obstructed DoorState = iota
	Closed
	Open
)

// doors handles the logic for opening and closing the elevator doors, as well as dealing with obstructions.
func doors(doorClosedCh chan<- bool, doorOpenCh <-chan bool, doorObstructedCh chan<- bool) {
	elevio.SetDoorOpenLamp(false)

	obstructionCh := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionCh)

	doorState := Closed
	obstruction := false
	doorTimer := time.NewTimer(time.Hour)
	doorTimer.Stop()

	for {
		select {

		// Handle door open requests:
		case <-doorOpenCh:

			switch doorState {
			case Open:
				doorTimer = time.NewTimer(config.DoorOpenDuration)
			case Closed:
				elevio.SetDoorOpenLamp(true)
				doorTimer = time.NewTimer(config.DoorOpenDuration)
				doorState = Open
			case Obstructed:
				doorTimer = time.NewTimer(config.DoorOpenDuration)
				doorState = Open
			default:
				panic("Invalid door state")
			}

		// Handle door obstruction status updates:
		case obstruction = <-obstructionCh:
			if doorState == Obstructed && !obstruction {
				doorTimer = time.NewTimer(config.DoorOpenDuration)
				doorState = Open
			}
			if obstruction {
				doorObstructedCh <- true
			} else {
				doorObstructedCh <- false
			}

		// Handle door close timer:
		case <-doorTimer.C:
			if obstruction {
				doorState = Obstructed
			} else {
				elevio.SetDoorOpenLamp(false)
				doorClosedCh <- true
				doorState = Closed
			}
		}
	}
}
