package distributor

import (
	"encoding/json"
	"fmt"
	"heis/config"
	"heis/elevator"
	"os"
)

type persistState struct {
	CabCalls  [config.NumFloors]bool
	Direction elevator.MotorDirection
}

func persistFilename(elevID int) string {
	return fmt.Sprintf("cab_orders_%d.json", elevID)
}

func SaveCabCalls(elevID int, calls [config.NumFloors]bool) {
	current := loadPersistState(elevID)
	current.CabCalls = calls
	savePersistState(elevID, current)
}

func SaveDirection(elevID int, dir elevator.MotorDirection) {
	current := loadPersistState(elevID)
	current.Direction = dir
	savePersistState(elevID, current)
}

func LoadCabCalls(elevID int) [config.NumFloors]bool {
	return loadPersistState(elevID).CabCalls
}

func LoadDirection(elevID int) elevator.MotorDirection {
	dir := loadPersistState(elevID).Direction
	if dir == elevator.Up || dir == elevator.Down {
		return dir
	}
	return elevator.Down // default
}

func savePersistState(elevID int, state persistState) {
	data, err := json.Marshal(state)
	if err != nil {
		fmt.Println("[cab_persistence] Failed to marshal state:", err)
		return
	}
	err = os.WriteFile(persistFilename(elevID), data, 0644)
	if err != nil {
		fmt.Println("[cab_persistence] Failed to write state to disk:", err)
	}
}

func loadPersistState(elevID int) persistState {
	var state persistState
	data, err := os.ReadFile(persistFilename(elevID))
	if err != nil {
		return state
	}
	err = json.Unmarshal(data, &state)
	if err != nil {
		fmt.Println("[cab_persistence] Failed to parse state from disk:", err)
	}
	return state
}
