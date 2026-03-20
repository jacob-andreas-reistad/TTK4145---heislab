package config

//Simple config for elevators
import (
	"time"
)

const (
	NumFloors       = 4
	NumElevators    = 3
	NumButtons      = 3
	PeersPortNumber = 58735
	BcastPortNumber = 58750
	Buffer          = 1024

	DisconnectTime   = 1 * time.Second
	DoorOpenDuration = 3 * time.Second
	WatchdogTime     = 3 * time.Second
	HeartbeatTime    = 15 * time.Millisecond
	InitSettleTime   = 50 * time.Millisecond
	MotorRetryTime   = 500 * time.Millisecond
)
