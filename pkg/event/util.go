package event

import (
	"fmt"
	"time"
)

func FormatTimeStr(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixNano()/int64(time.Millisecond))
}
