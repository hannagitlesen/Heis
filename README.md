# Elevator Project

## Project description
This repository creates software for controlling `n` elevators working in parallel across `m` floors.

## Solution

#### Communication
Our solution has a peer-to-peer architecture with a fleeting master. The master is the elevator that receives the local order, and it is responsible for assigning the order to the elevator with the lowest cost. With this architecture, all the elevators should always be up to date with the states and orders of the other elevators on the network. This is stored in a map that is denoted elevators. The state of an elevator contains the floor it is currently in, the behaviour, and the direction it is going. The orders are stored in a `m x n` matrix containing the states of the requests. A 0 donates no order, 1 denotes an order, and 2 denotes an confirmed order.

UDP is the communication protocol used in this solution. When an elevator distributes an order, the decision is broadcasted to the network. This solution will handle the event of network loss of a node, as the other elevators will handle the requests of the lost elevator. The lost elevator will work as an individual elevator.

### System

#### Config
Config includes all global variables, structs, and enums used in the other modules.

#### Assigner
The Assigner-module is responsible for finding the ID of the most suitable elevator for the order. It calculates a cost function based on the time until the elevators are back in idle. 

#### Distributor
The Distributor-module is responsible for distributing and synchronizing the connected elevators in the network. In summation it:
* Use the Assigner-module to select the most suitable elevator for the order
* Create channels for communication with the network. The elevators send and receive states and orders
* Checks for elevators that enters / exits the system
* Send orders to the local elevator



