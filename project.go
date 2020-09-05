package tb

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

var (
	ErrAlreadyStarted       = errors.New("project is already running")
	ErrProjectAlreadyExists = errors.New("a project with that name already exists")
)

// A Project keeps track of a list of children project and time segments, each
// representing a unit of tracked time.
type Project struct {
	Name       string `json:"name"`
	IsRunning  bool   `json:"is_running"`
	IsArchived bool   `json:"is_archived"`

	Entries []Entry `json:"entries"`

	Children []*Project `json:"children"`
}

// New creates a new project either as a direct child of this project or as
// a child of a more distant child. It returns true if the new project was
// created as a child project of the current project, and false if it was not.
func (p *Project) New(name string) (added bool, err error) {
	if name == p.Name {
		return false, ErrProjectAlreadyExists
	} else if strings.HasPrefix(name, p.Name+"/") {
		name = strings.TrimPrefix(name, p.Name+"/")

		for _, c := range p.Children {
			added, err = c.New(name)
			if added || err != nil {
				return added, err
			}
		}

		// Didn't fit any child, add it to this project
		newProj := &Project{Name: name}
		// If the new name still has /s then create those intermediary projects
		if split := strings.Split(name, "/"); len(split) > 1 {
			for i := len(split) - 1; i >= 0; i-- {
				newProj = &Project{Name: split[i], Children: []*Project{newProj}}
			}
		}

		p.Children = append(p.Children, newProj)
		return true, nil
	} else {
		return false, nil
	}
}

// Status prints whether or not a project is running currently.
func (p *Project) Status() {
	if p.IsRunning {
		dur := time.Now().Sub(p.Entries[len(p.Entries)-1].Start)
		fmt.Printf("%s is running (%s)\n", p.Name, dur.Truncate(time.Second).String())
	}

	for _, c := range p.Children {
		c.Status()
	}
}

// Start starts time tracking a project. It does not start its children.
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

// Stop stops time tracking a project. It does not stop its children.
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
	if p.IsRunning {
		p.Stop()
	}

	for _, c := range p.Children {
		c.Archive()
	}

	p.IsArchived = true

	fmt.Printf("archived \"%s\"\n", p.Name)
}

// Stats show how long a project and all its children have been running for the
// dur amount of time. The amount of time shown next to a project will be the
// sum of the time that it and all of its children have been running.
func (p *Project) Stats(w *tabwriter.Writer, padding string, dur time.Duration, durString string) {
	if !p.IsArchived {
		name := p.Name
		arrow := ""

		if name == "" {
			name = "Total"
		} else {
			arrow = "↳ "
		}

		dur := p.durationSince(time.Now().Add(-dur))
		fmt.Fprintf(
			w,
			"%s%s\t%s\n",
			padding+arrow,
			name,
			dur.Truncate(time.Second).String(),
		)
	}

	for _, c := range p.Children {
		c.Stats(w, padding+"  ", dur, durString)
	}
}

// Timecard prints all the entries for a project. It will print the entries
// of all of its child projects too.
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

// printTimecard prints the entries just for this project and not its child
// projects.
func (p *Project) printTimecard(w *tabwriter.Writer, since time.Duration, config Config) {
	println(p.Name)
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

func (p *Project) FindProjects(name string) (matches []*Project) {
	if name == p.Name || strings.HasSuffix(p.Name, name) {
		matches = append(matches, p)
	}

	if strings.HasPrefix(name, p.Name+"/") {
		trimmedName := strings.TrimPrefix(name, p.Name+"/")

		for _, c := range p.Children {
			matches = append(matches, c.FindProjects(trimmedName)...)
		}
	} else {
		for _, c := range p.Children {
			matches = append(matches, c.FindProjects(name)...)
		}
	}

	return matches
}

// recalculate the duration of all entries.
func (p *Project) RecalculateEntires() {
	for _, c := range p.Children {
		c.RecalculateEntires()
	}

	for i, _ := range p.Entries {
		p.Entries[i].CalculateDuration()
	}
}

func (p *Project) Sort() {
	sort.Slice(p.Children, func(i, j int) bool {
		return p.Children[i].Name < p.Children[j].Name
	})
}
