package tb

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

var (
	ErrAlreadyStarted  = errors.New("project is already running")
	ErrProjectNotFound = errors.New("no project with that name found")
)

// A Project keeps track of a list of time segments, each representing
// a unit of tracked time.
// Note that Projects are NOT hierarchical.  They are all flat with a pseudo-
// hierarchy created using "/".
type Project struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`

	IsRunning  bool `json:"is_running"`
	IsArchived bool `json:"is_archived"`
}

// Start starts time tracking a project
func (p *Project) Start() (err error) {
	if p.IsRunning {
		return ErrAlreadyStarted
	} else {
		e := Entry{Start: time.Now()}
		p.Entries = append(p.Entries, e)
		p.IsRunning = true

		fmt.Printf("started \"%s\" at %s\n", p.Name, e.Start.Format("15:04 EDT"))
	}

	return err
}

// Stop stops time tracking a project
func (p *Project) Stop() (err error) {
	if p.IsRunning {
		p.Entries[len(p.Entries)-1].End = time.Now()
		p.Entries[len(p.Entries)-1].CalculateDuration()
		p.IsRunning = false

		dur := p.Entries[len(p.Entries)-1].Duration
		fmt.Printf("stopped \"%s\" after a duration of %s\n", p.Name, dur.Truncate(time.Second).String())
	} else {
		return fmt.Errorf("project \"%s\" isn't running", p.Name)
	}

	return err
}

// Archive hides a project and all its children
func (p *Project) Archive() {
	p.IsRunning = false
	p.IsArchived = true

	/*
		// TODO can't do until projects are made hierarchical
		for _, c := range p.projects {
			if strings.Contains(c.Name, p.Name+"/") {
				c.Archive()
			}
		}
	*/

	fmt.Printf("archived \"%s\"\n", p.Name)
}

// Timecard prints all the entries for a project.
func (p *Project) Timecard(config Config, projects []Project) {
	since, sinceStr := parseTimeString(3)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Project\tDate\tStart\tEnd\tDuration\n")

	p.printTimecard(w, since, config, projects)
	for _, c := range projects {
		if strings.Contains(c.Name, p.Name+"/") && !c.IsArchived {
			c.printTimecard(w, since, config, projects)
		}
	}

	w.Flush()
	fmt.Println("-----------------------------------------------------")
	dur := p.durationSince(time.Now().Add(-since), projects).Truncate(time.Second)
	fmt.Printf("Total duration: %s in the past %s\n", dur, sinceStr)
}

func (p *Project) printTimecard(w *tabwriter.Writer, since time.Duration, config Config, projects []Project) {
	entries := p.entriesSince(time.Now().Add(-since))
	for _, e := range entries {
		date := e.Start.Format(config.DateFormat)
		start := e.Start.Format(config.TimeFormat)
		end := e.End.Format(config.TimeFormat)

		var dur string
		if e.End.IsZero() {
			dur = "Running"
		} else {
			dur = e.Duration.Truncate(time.Second).String()
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", p.Name, date, start, end, dur)
	}
}

func (p *Project) entriesSince(t time.Time) (entries []Entry) {
	for _, s := range p.Entries {
		if t.Before(s.Start) {
			entries = append(entries, s)
		}
	}

	return entries
}

// TODO Wait why do we take in a []Project?
func (p *Project) durationSince(t time.Time, projects []Project) (dur time.Duration) {
	// Calculate own time
	for _, s := range p.Entries {
		if s.End.IsZero() {
			// If an entry hasn't ended, just count the time from start to now
			dur += time.Now().Sub(s.Start)
		} else if t.Before(s.Start) {
			dur += s.Duration
		}
	}

	// Calculate time of all children (projects with name p.Name/*)
	for _, c := range projects {
		if strings.Contains(c.Name, p.Name+"/") && !c.IsArchived {
			dur += c.durationSince(t, projects)
		}
	}

	return dur
}
