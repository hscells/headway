package headway

import "time"

type Progress struct {
	Name            string  `form:"name"`
	CurrentProgress float64 `form:"current"`
	TotalProgress   float64 `form:"total"`
	Comment         string  `form:"comment"`
	LastUpdate      time.Time
}
