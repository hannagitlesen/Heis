module main

go 1.17

require localelevator v0.0.0
replace localelevator => ./localElevator

require distributor v0.0.0
replace distributor => ./distributor

require assigner v0.0.0
replace assigner => ./assigner