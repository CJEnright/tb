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

var (
	projects []Project
)

// TODO Maybe have some config in .tb.json? like l10n for military/ISO dates, format strings
func main() {
	var err error

	path := os.Getenv("HOME") + "/.tb.json"
	err = load(path)

	l := len(os.Args)
	if l == 1 {
		Status()
	} else if l == 3 {
		command := os.Args[1]
		projectName := os.Args[2]

		// TODO make commands case insensitive
		// TODO I don't like how we deal with projectName, the same thing is repeated in like every command func
		if command == "new" {
			err = New(projectName)
		} else if command == "start" {
			err = Start(projectName)
		} else if command == "stop" {
			err = Stop(projectName)
		} else if command == "timecard" {
			err = Timecard(projectName)
		} else if command == "archive" {
			err = Archive(projectName)
		} else if command == "s" {
			// TODO this one should be a toggle
		}
	}
	if err != nil {
		fmt.Println(err)
	}

	err = save(path)
	if err != nil {
		fmt.Println(err)
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

func (p *Project) EntriesSince(t time.Time) (entries []Entry) {
	for _, s := range p.Entries {
		if t.Before(s.Start) {
			entries = append(entries, s)
		}
	}

	return entries
}

func (p *Project) DurationSince(t time.Time) (d time.Duration) {
	// Calculate own time
	for _, s := range p.Entries {
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

func Timecard(projectName string) (err error) {
	/*
		Date 	Start End   Duration
		11/2  11:45 13:22 45m32s
		11/2  11:45 13:22 45m32s

		Total: 3hr2min over the past week
	*/
	index, err := FindProjectIndex(projectName)
	if err != nil {
		return err
	}
	p := projects[index]

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Date\tStart\tEnd\tDuration\n")

	// TODO user inputted time range
	entries := p.EntriesSince(time.Now().AddDate(0, 0, -7))
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
	durSince := p.DurationSince(time.Now().AddDate(0, 0, -7)).Truncate(time.Second)
	fmt.Printf("Total duration: %s\n", durSince)
	return err
}

func Status() {
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

// TODO make stats and status different
func Stats(projectName string) (err error) {
	if len(os.Args) < 3 {
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

	return err
}

func New(projectName string) (err error) {
	index := IndexOf(projects, projectName)
	if index != -1 {
		return fmt.Errorf("project with name \"%s\" already exists", projectName)
	}

	p := Project{Name: projectName}
	projects = append(projects, p)

	fmt.Printf("created project \"%s\"\n", p.Name)

	return err
}

func Start(projectName string) (err error) {
	index, err := FindProjectIndex(projectName)
	if err != nil {
		return err
	}

	projectName = projects[index].Name

	if projects[index].Active {
		return fmt.Errorf("project \"%s\" is already running", projectName)
	} else {
		e := Entry{Start: time.Now()}
		projects[index].Entries = append(projects[index].Entries, e)
		projects[index].Active = true

		fmt.Printf("started \"%s\" at %s\n", projectName, e.Start.Format("15:04 EDT"))
	}

	return err
}

func Stop(projectName string) (err error) {
	index, err := FindProjectIndex(projectName)
	if err != nil {
		return err
	}

	projectName = projects[index].Name

	if projects[index].Active {
		projects[index].Entries[len(projects[index].Entries)-1].End = time.Now()
		projects[index].Entries[len(projects[index].Entries)-1].FindDuration()
		projects[index].Active = false

		dur := projects[index].Entries[len(projects[index].Entries)-1].Duration
		fmt.Printf("stopped \"%s\" after a duration of %s\n", projects[index].Name, dur.Truncate(time.Second).String())
	} else {
		return fmt.Errorf("project \"%s\" isn't running", projectName)
	}

	return err
}

func Archive(projectName string) (err error) {
	index, err := FindProjectIndex(projectName)
	if err != nil {
		return err
	}

	projectName = projects[index].Name

	projects[index].Active = false
	projects[index].Archived = true
	fmt.Printf("Archived \"%s\"\n", projectName)

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

func IndexOfSuffix(a []Project, s string) (i int) {
	for i, e := range a {
		if strings.HasSuffix(e.Name, s) {
			return i
		}
	}

	return -1
}

func Contains(s []string, e string) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}

	return false
}

func FindProjectIndex(projectName string) (index int, err error) {
	index = IndexOf(projects, projectName)
	if index == -1 {
		index = IndexOfSuffix(projects, projectName)
	}
	if index == -1 {
		return index, fmt.Errorf("no project named \"%s\" found", projectName)
	}

	return index, nil
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
