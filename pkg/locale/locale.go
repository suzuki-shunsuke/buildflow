package locale

import "time"

func UTC() *time.Location {
	return time.FixedZone("UTC", 0)
}
