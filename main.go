// TODO marshal/unmarshal pointers doesn't work too well, take them out if possible
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	DefaultDateFormat = "01/02"
	DefaultTimeFormat = "15:04:05"
)

// TODO Maybe have some config in .tb.json? like l10n for 24h/ISO dates, format strings
// TODO stats and status
// TODO ideally we shouldn't be reading the whole json file on every operation
// because most of them will just be starting/stopping.  The only time we do
// need to read operations is when we're calculating times.  the same kinda
// goes with config.  However, I'd also like to keep everything to one file
// which makes this a bit more difficult.
func main() {
	path := os.Getenv("HOME") + "/.tb.json"
	tb, err := load(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	didEdit := false

	l := len(os.Args)
	if l == 1 {
		Status(tb)
	} else if l == 3 {
		// ToLower will make commands case insensitive
		command := strings.ToLower(os.Args[1])
		projectName := os.Args[2]

		var p *Project
		p, err = FindProject(tb, projectName)

		if command == "new" {
			if err == nil && p.Name == projectName {
				fmt.Printf("project with name \"%s\" already exists\n", p.Name)
				return
			} else {
				err = nil

				p := Project{Name: projectName}
				tb.Projects = append(tb.Projects, p)
				didEdit = true

				fmt.Printf("created project \"%s\"\n", p.Name)
			}
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		switch command {
		case "start":
			err = p.Start()
			didEdit = true
		case "stop":
			err = p.Stop()
			didEdit = true
		case "s":
			if p.Active == true {
				p.Stop()
			} else {
				p.Start()
			}
			didEdit = true
		case "timecard":
			err = p.PrintTimecard(tb.Conf)
		case "archive":
			p.Archive()
			didEdit = true
		}
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	if didEdit {
		err = save(path, tb)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

type tbWrapper struct {
	Conf     Config    `json:"config"`
	Projects []Project `json:"projects"`
}

type Config struct {
	DateFormat string `json:"dateFormat"`
	TimeFormat string `json:"timeFormat"`
}

type Entry struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

func (s *Entry) FindDuration() {
	s.Duration = s.End.Sub(s.Start)
}

type Project struct {
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	Archived  bool      `json:"archived"`
	StartedAt time.Time `json:"startTime"`
	Entries   []Entry   `json:"segments"`
}

func (p *Project) Start() (err error) {
	if p.Active {
		return fmt.Errorf("project \"%s\" is already running", p.Name)
	} else {
		e := Entry{Start: time.Now()}
		p.Entries = append(p.Entries, e)
		p.Active = true

		fmt.Printf("started \"%s\" at %s\n", p.Name, e.Start.Format("15:04 EDT"))
	}

	return err
}

func (p *Project) Stop() (err error) {
	if p.Active {
		p.Entries[len(p.Entries)-1].End = time.Now()
		p.Entries[len(p.Entries)-1].FindDuration()
		p.Active = false

		dur := p.Entries[len(p.Entries)-1].Duration
		fmt.Printf("stopped \"%s\" after a duration of %s\n", p.Name, dur.Truncate(time.Second).String())
	} else {
		return fmt.Errorf("project \"%s\" isn't running", p.Name)
	}

	return err
}

func (p *Project) Archive() {
	p.Active = false
	p.Archived = true

	fmt.Printf("archived \"%s\"\n", p.Name)
}

func (p *Project) PrintTimecard(config Config) (err error) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Date\tStart\tEnd\tDuration\n")

	// TODO user inputted time range
	entries := p.entriesSince(time.Now().AddDate(0, 0, -7))
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

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", date, start, end, dur)
	}
	w.Flush()

	fmt.Println("---------------------------------------")
	// TODO user inputted time range
	durSince := p.durationSince(time.Now().AddDate(0, 0, -7)).Truncate(time.Second)
	// TODO have this say how far back duration goes
	fmt.Printf("Total duration: %s\n", durSince)
	return err
}

func (p *Project) entriesSince(t time.Time) (entries []Entry) {
	for _, s := range p.Entries {
		if t.Before(s.Start) {
			entries = append(entries, s)
		}
	}

	return entries
}

func (p *Project) durationSince(t time.Time) (d time.Duration) {
	// Calculate own time
	for _, s := range p.Entries {
		if s.End.IsZero() {
			// If something hasn't ended, just count the time from start to now
			d += time.Now().Sub(s.Start)
		} else if t.Before(s.Start) {
			d += s.Duration
		}
	}

	/*
		// Now calculate time of all children (projects with name p.Name/*)
		for _, c := range projects {
			if strings.Contains(c.Name, p.Name+"/") {
				d += c.durationSince(t)
			}
		}
	*/

	return d
}

func Status(tb *tbWrapper) {
	// TODO fix all this ish
	for _, p := range tb.Projects {
		name := p.Name
		var spacing string

		slashes := strings.Count(p.Name, "/")
		for i := 1; i < slashes; i++ {
			spacing += "  "
		}

		if slashes > 0 {
			spacing += "↳ "

			name = name[strings.LastIndex(name, "/")+1 : len(name)]
		}

		fmt.Printf("%s%s is %s\n", spacing, name, "running")
		/*
			if p.Active && !p.Archived {
				fmt.Printf("%s%s is %s\n", spacing, name, "running")
			} else if (*flagAllAndArchived || *flagAll) && !p.Archived {
				fmt.Printf("%s%s is %s\n", spacing, name, "not running")
			} else if *flagAllAndArchived && p.Archived {
				fmt.Printf("%s%s %s is %s\n", spacing, "[ARCHIVED]", name, "not running")
			}
		*/
	}
}

// TODO clean this up
/*
func Stats(projectName string) (err error) {
	if len(os.Args) < 3 {
	} else if len(os.Args) == 3 {
		if os.Args[2] == "day" {
			for _, p := range projects {
				dur := p.durationSince(time.Now().AddDate(0, 0, -1))
				fmt.Printf(
					"logged %s for %s in the past day\n",
					p.Name,
					dur.Truncate(time.Second).String(),
				)
			}
		} else if os.Args[2] == "week" {
			for _, p := range projects {
				dur := p.durationSince(time.Now().AddDate(0, 0, -7))
				fmt.Printf(
					"logged %s for %s in the past week\n",
					p.Name,
					dur.Truncate(time.Second).String(),
				)
			}
		} else if os.Args[2] == "month" {
			for _, p := range projects {
				dur := p.durationSince(time.Now().AddDate(0, -1, 0))
				fmt.Printf(
					"logged %s for %s in the past month\n",
					p.Name,
					dur.Truncate(time.Second).String(),
				)
			}
		}
	}

	return err
}
*/

// FindProject finds the first full match or suffix match of a project
// TODO check for collisions when suffix matching?
func FindProject(tb *tbWrapper, projectName string) (project *Project, err error) {
	index := -1

	for i, e := range tb.Projects {
		if e.Name == projectName {
			index = i
			break
		} else if strings.HasSuffix(e.Name, projectName) && index == -1 {
			index = i
		}
	}

	if index == -1 {
		return project, fmt.Errorf("no project named \"%s\" found", projectName)
	}

	return &tb.Projects[index], err
}

func load(path string) (tb *tbWrapper, err error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return tb, err
		}

		f.Write([]byte("{}"))
		f.Close()
	}

	f, err := os.Open(path)
	if err != nil {
		return tb, err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return tb, err
	}

	err = json.Unmarshal(bytes, &tb)
	if err != nil {
		return tb, err
	}

	if tb.Conf.DateFormat == "" {
		tb.Conf.DateFormat = DefaultDateFormat
	}

	if tb.Conf.TimeFormat == "" {
		tb.Conf.TimeFormat = DefaultTimeFormat
	}

	return tb, err
}

func save(path string, tb *tbWrapper) (err error) {
	// Always sort projects by name before writing them
	sort.Slice(tb.Projects, func(i, j int) bool {
		return tb.Projects[i].Name < tb.Projects[j].Name
	})

	out, err := json.Marshal(*tb)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, out, 0644)
	return err
}

func Contains(s []string, e string) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}

	return false
}
