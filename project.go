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
type Project struct {
	Name    string  `json:"name"` // Full path to the project, including parent names
	Entries []Entry `json:"entries"`

	Children []Project `json:"children"`

	IsRunning  bool `json:"is_running"`
	IsArchived bool `json:"is_archived"`
}

// Status prints whether or not a project is running currently.
func (p *Project) Status() {
	if p.IsRunning {
		fmt.Printf("%s is running\n", p.Name)
	}

	for _, c := range p.Children {
		c.Status()
	}
}

// Start starts time tracking a project.  Does not start its children.
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

// Stop stops time tracking a project.  Does not stop its children.
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

// Archive hides a project and all its children.
func (p *Project) Archive() {
	p.IsRunning = false
	p.IsArchived = true

	for _, c := range p.Children {
		c.Archive()
	}

	fmt.Printf("archived \"%s\"\n", p.Name)
}

func (p *Project) Stats(w *tabwriter.Writer, padding string, dur time.Duration, durString string) {
	if !p.IsArchived {
		name := p.Name
		if padding != "" {
			padding += "↳ "
			name = name[strings.LastIndex(name, "/")+1 : len(name)]
		}

		dur := p.durationSince(time.Now().Add(-dur))
		fmt.Fprintf(
			w,
			"%s%s\t%s\n",
			padding,
			name,
			dur.Truncate(time.Second).String(),
		)
	}

	for _, c := range p.Children {
		c.Stats(w, padding+"  ", dur, durString)
	}
}

// Timecard prints all the entries for a project.
func (p *Project) Timecard(config Config) {
	since, sinceStr := parseTimeString(3)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Project\tDate\tStart\tEnd\tDuration\n")

	p.printTimecard(w, since, config)
	for _, c := range p.Children {
		if !c.IsArchived {
			c.printTimecard(w, since, config)
		}
	}

	w.Flush()
	fmt.Println("-----------------------------------------------------")
	dur := p.durationSince(time.Now().Add(-since)).Truncate(time.Second)
	fmt.Printf("Total duration: %s in the past %s\n", dur, sinceStr)
}

func (p *Project) printTimecard(w *tabwriter.Writer, since time.Duration, config Config) {
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

// entriesSince returns the entries since a specific time.  It doesn't return the
// entries for its children.
func (p *Project) entriesSince(t time.Time) (entries []Entry) {
	for _, s := range p.Entries {
		if t.Before(s.Start) {
			entries = append(entries, s)
		}
	}

	return entries
}

// durationSince calculates the duration a project and all its children have been
// running for since a specific date.
func (p *Project) durationSince(t time.Time) (dur time.Duration) {
	// Calculate own time
	for _, s := range p.Entries {
		if s.End.IsZero() {
			// If an entry hasn't ended, just count the time from start to now
			dur += time.Now().Sub(s.Start)
		} else if t.Before(s.Start) {
			dur += s.Duration
		}
	}

	for _, c := range p.Children {
		if !c.IsArchived {
			dur += c.durationSince(t)
		}
	}

	return dur
}

func (p *Project) findProject(name string) []*Project {
	// Invariant: no two projects should ever have the exact same full path
	// name.  This is enforced by tbWrapper.New()

	// Keeps both exact and suffix matches
	matches := []*Project{}
	for _, c := range p.Children {
		c.findProject(name)
		matches = append(matches, c.findProject(name)...)
	}

	if p.Name == name || strings.HasSuffix(p.Name, name) {
		matches = append(matches, p)
	}

	return matches
}
