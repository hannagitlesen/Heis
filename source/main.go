package main

import (
	"fmt"
	le "localelevator"
)

func main() {
	e := le.NewElevator()
	fmt.Printf("%+v\n", e)
}
