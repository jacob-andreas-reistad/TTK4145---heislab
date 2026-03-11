// This file contains the logic for the elevator doors in the elevator system... tbc

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

func doors(doorClosedCh chan<- bool, doorOpenCh <-chan bool, doorObstructedCh chan<- bool) {
	elevio.SetDoorOpenLamp(false)

	obstructionCh := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionCh)

	door_state := Closed
	obstruction := false
	time_counter := time.NewTimer(time.Hour)
	time_counter.Stop()

	for {
		select {

		case <-doorOpenCh:
			if obstruction {
				obstructionCh <- true
			}
			switch door_state {
			case Open:
				time_counter = time.NewTimer(config.DoorOpenDuration)
			case Closed:
				elevio.SetDoorOpenLamp(true)
				time_counter = time.NewTimer(config.DoorOpenDuration)
				door_state = Open
			case Obstructed:
				time_counter = time.NewTimer(config.DoorOpenDuration)
				door_state = Open
			default:
				panic("Invalid door state")
			}

		case obstruction = <-obstructionCh:
			if door_state == Obstructed && !obstruction {
				elevio.SetDoorOpenLamp(false)
				doorClosedCh <- true
				door_state = Closed
			}
			if obstruction {
				doorObstructedCh <- true
			} else {
				doorObstructedCh <- false
			}

		case <-time_counter.C:
			if door_state == Open {
				panic("Door state undefined")
			}
			if obstruction {
				door_state = Obstructed
			} else {
				elevio.SetDoorOpenLamp(false)
				doorClosedCh <- true
				door_state = Closed
			}
		}
	}
}
