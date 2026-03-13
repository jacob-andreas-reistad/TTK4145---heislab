package assigner

import (
	"encoding/json"
	"fmt"
	"heis/config"
	"heis/distributor"
	"heis/elevator"
	"os/exec"
	"runtime"
	"strconv"
)

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	Behavior    string                 `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Direction   string                 `json:"direction"`
	CabRequests [config.NumFloors]bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [config.NumFloors][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState   `json:"states"`
}

func CostFunction(cs distributor.CommonState, ID int) elevator.Order {
	ElevStates := make(map[string]HRAElevState)
	for i, j := range cs.Elevators {
		if cs.Acks[i] == distributor.Disconnected || j.Current.MotorStop || j.Current.Obstructed {
			continue
		} else {
			ElevStates[strconv.Itoa(i)] = HRAElevState{
				Behavior:    j.Current.Behaviour.ToString(),
				Floor:       j.Current.Floor,
				Direction:   j.Current.Direction.ToString(),
				CabRequests: j.CabCalls,
			}
		}
	}

	Input := HRAInput{cs.HallCalls, ElevStates}

	Executable := ""
	switch runtime.GOOS {
	case "linux":
		Executable = "hall_request_assigner"
	case "windows":
		Executable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	jsonBytes, err := json.Marshal(Input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		panic("json.Marshal error: ")
	}

	ret, err := exec.Command("assigner/executables/"+Executable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		panic("exec.Command error: ")
	}

	output := new(map[string]elevator.Order)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		panic("json.Unmarshal error: ")
	}
	return (*output)[strconv.Itoa(ID)]
}
