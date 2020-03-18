package tb

import "time"

type Entry struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

func (e *Entry) CalculateDuration() {
	e.Duration = e.End.Sub(e.Start)
}
