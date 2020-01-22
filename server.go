package headway

import "time"

type Progress struct {
	RateEstimate    float64
	CurrentProgress float64 `form:"current"`
	TotalProgress   float64 `form:"total"`
	Name            string  `form:"name"`
	Comment         string  `form:"comment"`
	Remaining       string
	Elapsed         string
	LastTook        string
	LastCompleted   time.Duration
	Started         time.Time
	LastUpdate      time.Time
}
