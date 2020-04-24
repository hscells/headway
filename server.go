package headway

import "time"

type Progress struct {
	LastCompleted   time.Duration            // Computed duration for how long the last item took.
	RateEstimate    float64                  // Computed Estimate of how many seconds per update the time estimate should increase/decrease.
	CurrentProgress float64 `form:"current"` // Current progress of the task.
	TotalProgress   float64 `form:"total"`   // Total amount of progress of the task.
	Name            string  `form:"name"`    // Identifier of the task.
	Comment         string  `form:"comment"` // Additional information, related to the task.
	Message         string  `form:"message"` // Message to pass into slack.
	Secret          string  `form:"Secret"`  // Client Secret to authenticate log requests.
	User            string                   // Computed username from slack.
	Remaining       string                   // Computed string for time remaining.
	Elapsed         string                   // Computed string for time elapsed.
	LastTook        string                   // Computed string for how much time the last item took.
	Started         time.Time                // Computed duration for when the task was started.
	LastUpdate      time.Time                // Computed time the last task was last updated.
}
