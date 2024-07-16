package time

import "time"

// DBNow just to make sure the time is consistent regardless of what time zone is local
func DBNow() time.Time {
	return time.Now().UTC()
}
