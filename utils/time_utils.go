package utils

import "math"

func SecondsToMinAndSec(seconds int64) (int, int) {
	minutes := math.Floor(float64(seconds) / 60)
	remainingSeconds := int(seconds) % 60
	return int(minutes), remainingSeconds
}

// TODO: WTF is this, a different kind of rounding?
func IntSecondsToMinAndSec(seconds int) (int, int) {
	minutes := seconds / 60
	remainingSeconds := seconds % 60
	return minutes, remainingSeconds
}
