# Elevator Project

## Project description
This repository creates software for controlling `n` elevators working in parallel across `m` floors.

## Solution

#### Communication
Our solution has a peer-to-peer architecture with a fleeting master. The fleeting master is the elevator that receives the local order, and it is responsible for assigning the order to the elevator with the lowest cost. With a peer-to-peer architecture, all the elevators on the network should always be up to date on the states and active orders of the others. The states and orders of each elevator are stored in a map that is denoted elevators. The state of an elevator contains the floor it is currently in, its behaviour, and the direction it is moving in. The orders are stored in a `m x n` matrix containing the states of the requests. A 0 denotes no order, 1 denotes an unconfirmed order, and 2 denotes a confirmed order.

UDP is the communication protocol used in this solution. When an elevator distributes an order, the decision is broadcasted to the network. In addition to this, each elevator regularly broadcasts an update of their state. The use of UDP in combination with a peer-to-peer solution will handle the event of network loss of a node. In this case, the hall-orders of the lost elevator will be assigned to the other elevators as they have a copy of all the currently active orders. The lost elevator will work as an individual elevator. Backup and restore of orders in case of network loss or power loss is easy to handle with this solution, as all elevators store the last known state of every other elevator.

### System

#### Config
Config includes all global variables, structs, and enums used in the other modules. 

#### Assigner
The Assigner-module is responsible for finding the ID of the most suitable elevator for the order. It utilizies a function for calculating the time it would take the elevators to return to an idle state if they were to execute the order. 

#### Watchdog
Watchdog is used to monitor if any of the elevators enters an unavailable state.

#### Local elevator
The localElevator-module contains the local FSM for the single elevator.

#### Distributor
The Distributor-module is responsible for distributing and synchronizing the connected elevators on the network. In summation it:
* Uses the Assigner-module to select the most suitable elevator for the order.
* Creates channels for communication with the network. The elevators send and receive states and orders.
* Checks for elevators that enter / exit the system.
* Sends orders to the local elevator.
* Monitors which elevators are in an unavailable state.






