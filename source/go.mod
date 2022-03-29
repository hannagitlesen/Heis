module main

go 1.17

require localelevator v0.0.0

replace localelevator => ./localElevator

require distributor v0.0.0

replace distributor => ./distributor

require assigner v0.0.0

replace assigner => ./assigner

require config v0.0.0

replace config => ./config

require watchdog v0.0.0

replace watchdog => ./watchdog

require bcast v0.0.0

replace bcast => ./network/bcast

require conn v0.0.0

replace conn => ./network/conn

require localip v0.0.0

replace localip => ./network/localip

require peers v0.0.0

replace peers => ./network/peers
