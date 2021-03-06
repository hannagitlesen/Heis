package config

import (
	"elevio"
)

const NumFloors = 4
const NumButtons = 3
const TravelTime = 5
const PeersPort = 15647
const BcastPort = 16569
const DoorTimerDuration = 3
const WatchdogTimeout = 4
const ConnectTimeout = 2
const LocalStateUpdate = 1
const BcastStateUpdate = 100

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
)

type DistributorElevator struct {
	Floor     int
	Direction MotorDirection
	Requests  [][]RequestsState
	Behaviour ElevBehaviour
}

type MessageType int

const (
	Order      MessageType = 0
	ElevStatus             = 1
)

type OrderMessage struct {
	AssignedID string
	Order      elevio.ButtonEvent
}

type BroadcastMessage struct {
	SenderID       string
	MsgType        MessageType
	ElevsStatusMsg map[string]DistributorElevator 
	OrderMsg       OrderMessage
}
