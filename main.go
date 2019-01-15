package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// TODO Maybe have some config in .tb.json? like l10n for 24h/ISO dates, format strings
// TODO stats and status
// TODO ideally we shouldn't be reading the whole json file on every operation
// because most of them will just be starting/stopping.  The only time we do
// need to read operations is when we're calculating times.
func main() {
	path := os.Getenv("HOME") + "/.tb.json"
	projects, err := loadProjects(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	l := len(os.Args)
	if l == 1 {
		Status(projects)
	} else if l == 3 {
		// ToLower will make commands case insensitive
		command := strings.ToLower(os.Args[1])
		projectName := os.Args[2]

		var p *Project
		p, err = FindProject(projects, projectName)

		if command == "new" {
			if err == nil && p.Name == projectName {
				fmt.Printf("project with name \"%s\" already exists\n", p.Name)
				return
			} else {
				err = nil

				p := Project{Name: projectName}
				projects = append(projects, p)
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
		case "stop":
			err = p.Stop()
		case "s":
			if p.Active == true {
				p.Stop()
			} else {
				p.Start()
			}
		case "timecard":
			err = p.PrintTimecard()
		case "archive":
			p.Archive()
		}
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	err = saveProjects(path, projects)
	if err != nil {
		fmt.Println(err)
		return
	}
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

func (p *Project) PrintTimecard() (err error) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Date\tStart\tEnd\tDuration\n")

	// TODO user inputted time range
	entries := p.entriesSince(time.Now().AddDate(0, 0, -7))
	for _, e := range entries {
		date := strconv.Itoa(int(e.Start.Month())) + "/" + strconv.Itoa(e.Start.Day())
		// TODO these things are wacci af
		start := strconv.Itoa(e.Start.Hour()) + ":" + strconv.Itoa(e.Start.Minute()) + ":" + strconv.Itoa(e.Start.Second())
		end := strconv.Itoa(e.End.Hour()) + ":" + strconv.Itoa(e.End.Minute()) + ":" + strconv.Itoa(e.End.Second())

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

func Status(projects []Project) {
	// TODO fix all this ish
	for _, p := range projects {
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
func FindProject(projects []Project, projectName string) (project *Project, err error) {
	index := -1

	for i, e := range projects {
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

	return &projects[index], err
}

func loadProjects(path string) (projects []Project, err error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(input, &projects)
	return projects, err
}

func saveProjects(path string, projects []Project) (err error) {
	// Always sort projects by name before writing them
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	out, err := json.Marshal(projects)
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
