package localelevator

import (
	"time"
)

func InitTimers() {
	doorTimer := time.NewTimer(time.Duration(DoorTimerDuration) * time.Second)
	
}