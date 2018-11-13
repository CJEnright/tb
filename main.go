package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	clr "github.com/logrusorgru/aurora"
)

type Segment struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

type Project struct {
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	Archived  bool      `json:"archived"`
	StartedAt time.Time `json:"startTime"`
	Segments  []Segment `json:"segments"`
}

func (s *Segment) CalculateDuration() {
	s.Duration = s.End.Sub(s.Start)
}

func (p *Project) DurationSince(t time.Time) (d time.Duration) {
	// Calculate own time
	for _, s := range p.Segments {
		if s.End.IsZero() {
			// If something hasn't ended, just count the time from start to now
			d += time.Now().Sub(s.Start)
		} else if t.Before(s.Start) {
			d += s.Duration
		}
	}

	// Now calculate time of all children (projects with name p.Name/*)
	for _, c := range projects {
		if strings.Contains(c.Name, p.Name+"/") {
			d += c.DurationSince(t)
		}
	}

	return d
}

var (
	projects []Project
)

func main() {
	var err error

	path := os.Getenv("HOME") + "/.tb.json"
	err = load(path)

	l := len(os.Args)
	if l == 1 {
		stats()
	} else if l >= 2 {
		if os.Args[1] == "new" {
			err = create()
		} else if os.Args[1] == "start" {
			err = start()
		} else if os.Args[1] == "stop" {
			err = stop()
		} else if os.Args[1] == "archive" {
			err = archive()
		} else {
			stats()
		}
	}

	err = save(path)

	if err != nil {
		fmt.Println(err)
	}
}

func load(path string) (err error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(input, &projects)
	return err
}

func save(path string) (err error) {
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

func stats() {
	flagAll := flag.Bool("a", false, "Show timers not currently running")
	flagAllAndArchived := flag.Bool("A", false, "Shows all timers, even archived ones")
	flag.Parse()

	if len(os.Args) < 3 {
		for _, p := range projects {
			name := p.Name
			var spacing string

			if *flagAll {
				slashes := strings.Count(p.Name, "/")
				for i := 1; i < slashes; i++ {
					spacing += "  "
				}

				if slashes > 0 {
					spacing += "↳ "

					name = name[strings.LastIndex(name, "/")+1 : len(name)]
				}
			}

			if p.Active && !p.Archived {
				fmt.Printf(
					"%s%s is %s\n",
					spacing,
					clr.Bold(name),
					clr.Green("running"),
				)
			} else if (*flagAllAndArchived || *flagAll) && !p.Archived {
				fmt.Printf(
					"%s%s is %s\n",
					spacing,
					clr.Bold(name),
					clr.Red("not running"),
				)
			} else if *flagAllAndArchived && p.Archived {
				fmt.Printf(
					"%s%s %s is %s\n",
					spacing,
					clr.Bold("[ARCHIVED]"),
					clr.Bold(name),
					clr.Red("not running"),
				)
			}
		}
	} else if len(os.Args) == 3 {
		if os.Args[2] == "day" {
			for _, p := range projects {
				dur := p.DurationSince(time.Now().AddDate(0, 0, -1))
				fmt.Printf(
					"logged %s for %s in the past day\n",
					p.Name,
					dur.Truncate(time.Second).String(),
				)
			}
		} else if os.Args[2] == "week" {
			for _, p := range projects {
				dur := p.DurationSince(time.Now().AddDate(0, 0, -7))
				fmt.Printf(
					"logged %s for %s in the past week\n",
					p.Name,
					dur.Truncate(time.Second).String(),
				)
			}
		} else if os.Args[2] == "month" {
			for _, p := range projects {
				dur := p.DurationSince(time.Now().AddDate(0, -1, 0))
				fmt.Printf(
					"logged %s for %s in the past month\n",
					p.Name,
					dur.Truncate(time.Second).String(),
				)
			}
		}
	}
}

func create() (err error) {
	var projectName string

	if len(os.Args) < 3 {
		fmt.Print("Enter project name: ")
		fmt.Scanln(&projectName)
		for IndexOf(projects, projectName) != -1 {
			fmt.Printf("Project name \"%s\" is already taken\n", projectName)
			fmt.Print("Enter project name: ")
			fmt.Scanln(&projectName)
		}

		p := Project{Name: projectName}
		projects = append(projects, p)

		fmt.Printf("Created \"%s\"\n", p.Name)

		return err
	} else {
		projectName = os.Args[2]

		if IndexOf(projects, projectName) == -1 {
			p := Project{Name: projectName}
			projects = append(projects, p)

			fmt.Printf("Created \"%s\"\n", p.Name)
		} else {
			return fmt.Errorf("Project already exists with name \"%s\"", projectName)
		}
	}

	return err
}

func start() (err error) {
	if len(os.Args) < 3 {
		return fmt.Errorf("No project specified")
	}

	projectName := os.Args[2]

	index := IndexOf(projects, projectName)
	if index != -1 {
		if projects[index].Active {
			return fmt.Errorf("Project \"%s\" is already running", projectName)
		} else {
			p := projects[index]

			seg := Segment{Start: time.Now()}
			p.Segments = append(p.Segments, seg)
			p.Active = true

			projects[index] = p

			fmt.Printf("Started \"%s\" at %s\n", projectName, seg.Start.Format("15:04 EDT"))
		}
	} else {
		var yn string
		fmt.Printf("No project named \"%s\" found, would you like to make it? (y/n)\n", projectName)

		fmt.Scanln(&yn)
		for yn != "y" && yn != "n" {
			fmt.Print("(y/n)")
			fmt.Scanln(&yn)
		}

		if yn == "y" {
			create()
			start()
		}

	}

	return err
}

func stop() (err error) {
	if len(os.Args) < 3 {
		return fmt.Errorf("No project specified")
	}

	projectName := os.Args[2]

	index := IndexOf(projects, projectName)
	if index != -1 {
		if projects[index].Active {
			p := projects[index]

			p.Active = false
			p.Segments[len(p.Segments)-1].End = time.Now()
			p.Segments[len(p.Segments)-1].CalculateDuration()

			projects[index] = p

			dur := p.Segments[len(p.Segments)-1].Duration
			fmt.Printf("Stopped \"%s\" after a duration of %s\n", p.Name, dur.Truncate(time.Second).String())
		} else {
			return fmt.Errorf("Project \"%s\" isn't running", projectName)
		}
	} else {
		// Not offering to create a project here for UX-y reasons
		// In most cases reaching here means you had a typo
		return fmt.Errorf("No project named \"%s\" found", projectName)
	}

	return err
}

func archive() (err error) {
	if len(os.Args) < 3 {
		return fmt.Errorf("No project specified")
	}

	projectName := os.Args[2]

	index := IndexOf(projects, projectName)
	if index != -1 {
		projects[index].Active = false
		projects[index].Archived = true
		fmt.Printf("Archived \"%s\"\n", projectName)
	} else {
		return fmt.Errorf("No project named \"%s\" found", projectName)
	}

	return err
}

func IndexOf(a []Project, s string) (i int) {
	for i, e := range a {
		if e.Name == s {
			return i
		}
	}

	return -1
}
