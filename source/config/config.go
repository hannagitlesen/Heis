package config

import (
	"fmt"
	"localip"
	"os"
)

const NumFloors = 4
const NumButtons = 3
const DoorTimerDuration = 3
const TravelTime = 5
const PeersPort = 15647
const BcastPort = 16569
const FailureTimeout = 10
const ConnectTimeout = 5
const UpdateTimeout = 3

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down                = -1
	MD_Stop                = 0
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown            = 1
	BT_Cab                 = 2
)

type ElevBehaviour int

const (
	Idle        ElevBehaviour = 0
	DoorOpen    ElevBehaviour = 1
	Moving      ElevBehaviour = 2
	Unavailable ElevBehaviour = 3
)

type RequestsState int

const (
	None        RequestsState = 0
	Unconfirmed RequestsState = 1
	Confirmed   RequestsState = 2
	Completed   RequestsState = 3
)

type DistributorElevator struct {
	Floor     int
	Direction MotorDirection
	Requests  [][]RequestsState
	Behaviour ElevBehaviour
}

func GetLocalIP() string {
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}
	id := fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	return id
}
