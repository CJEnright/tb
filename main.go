package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	DefaultDateFormat = "01/02"
	DefaultTimeFormat = "15:04:05"
)

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
	} else if l == 2 {
		command := strings.ToLower(os.Args[1])

		switch command {
		case "stats":
			Stats(tb.Projects)
		}
	} else {
		// ToLower will make commands case insensitive
		command := strings.ToLower(os.Args[1])
		projectName := os.Args[2]

		var p *Project
		p, err = FindProject(tb, projectName)

		switch command {
		case "new":
			if err == nil && p.Name == projectName {
				fmt.Printf("project with name \"%s\" already exists\n", p.Name)
				return
			} else {
				// Add base projects so stats come out right
				roots := strings.Split(projectName, "/")
				for i := 0; i < len(roots)-1; i++ {
					combined := strings.Join(roots[:i+1], "/")
					// Make sure project with this name doesn't exist
					_, err = FindProject(tb, combined)
					if err != nil {
						p := Project{Name: combined}
						tb.Projects = append(tb.Projects, p)
					}
				}

				p := Project{Name: projectName}
				tb.Projects = append(tb.Projects, p)
				didEdit = true

				fmt.Printf("created project \"%s\"\n", p.Name)
				err = nil
			}
		case "stats":
			Stats(tb.Projects)
			err = nil
			break
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
			p.Timecard(tb.Conf, tb.Projects)
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

func (p *Project) Timecard(config Config, projects []Project) {
	since := ParseTimeString(3)

	var sinceString string
	if len(os.Args) <= 3 {
		sinceString = "week"
	} else {
		sinceString = strings.Join(os.Args[3:], " ")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Project\tDate\tStart\tEnd\tDuration\n")

	p.PrintTimecard(w, since, config, projects)
	for _, c := range projects {
		if strings.Contains(c.Name, p.Name+"/") && !c.Archived {
			c.PrintTimecard(w, since, config, projects)
		}
	}

	w.Flush()
	fmt.Println("-----------------------------------------------------")
	dur := p.durationSince(time.Now().Add(-since), projects).Truncate(time.Second)
	fmt.Printf("Total duration: %s in the past %s\n", dur, sinceString)
}

// TODO have this count children and add a name column
func (p *Project) PrintTimecard(w *tabwriter.Writer, since time.Duration, config Config, projects []Project) {
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

func (p *Project) durationSince(t time.Time, projects []Project) (d time.Duration) {
	// Calculate own time
	for _, s := range p.Entries {
		if s.End.IsZero() {
			// If something hasn't ended, just count the time from start to now
			d += time.Now().Sub(s.Start)
		} else if t.Before(s.Start) {
			d += s.Duration
		}
	}

	// Calculate time of all children (projects with name p.Name/*)
	for _, c := range projects {
		if strings.Contains(c.Name, p.Name+"/") && !c.Archived {
			d += c.durationSince(t, projects)
		}
	}

	return d
}

func Status(tb *tbWrapper) {
	for _, p := range tb.Projects {
		if p.Active {
			fmt.Printf("%s is %s\n", p.Name, "running")
		}
	}
}

func Stats(projects []Project) (err error) {
	var since time.Duration
	if len(os.Args) < 3 {
		since = time.Since(time.Now().AddDate(0, 0, -7))
	} else {
		since = ParseTimeString(2)
	}

	for _, p := range projects {
		if !p.Archived {
			var spacing string
			slashes := strings.Count(p.Name, "/")
			for i := 1; i < slashes; i++ {
				spacing += "  "
			}

			name := p.Name
			if slashes > 0 {
				spacing += "↳ "

				name = name[strings.LastIndex(name, "/")+1 : len(name)]
			}

			dur := p.durationSince(time.Now().Add(-since), projects)
			fmt.Printf(
				"%s%s for %s in the past %s\n",
				spacing,
				name,
				dur.Truncate(time.Second).String(),
				strings.Join(os.Args[2:], " "),
			)
		}
	}

	return err
}

// FindProject finds the first full match or suffix match of a project
func FindProject(tb *tbWrapper, projectName string) (project *Project, err error) {
	var potentialIndexes []int

	for i, e := range tb.Projects {
		if e.Name == projectName {
			return &tb.Projects[i], err
		} else if strings.HasSuffix(e.Name, projectName) {
			potentialIndexes = append(potentialIndexes, i)
		}
	}

	if len(potentialIndexes) > 1 {
		response := 0
		for response < 1 || response > len(potentialIndexes) {
			fmt.Printf("multiple projects found with suffix \"%s\":\n", projectName)

			for i, v := range potentialIndexes {
				fmt.Printf("(%d) %s\n", i+1, tb.Projects[v].Name)
			}

			_, err := fmt.Scanln(&response)
			if err != nil {
				return project, err
			}

			return &tb.Projects[potentialIndexes[response-1]], err
		}
	} else if len(potentialIndexes) == 1 {
		return &tb.Projects[potentialIndexes[0]], err
	}

	return project, fmt.Errorf("no project named \"%s\" found", projectName)
}

func ParseTimeString(startIndex int) (dur time.Duration) {
	if len(os.Args) <= startIndex {
		return time.Since(time.Now().AddDate(0, 0, -7))
	}

	timeString := strings.ToLower(strings.Join(os.Args[startIndex:], " "))
	switch {
	case strings.Contains(timeString, "hour"):
		dur = time.Since(time.Now().Add(-1 * time.Hour))
	case strings.Contains(timeString, "day"):
		dur = time.Since(time.Now().AddDate(0, 0, -1))
	case strings.Contains(timeString, "week"):
		dur = time.Since(time.Now().AddDate(0, 0, -7))
	case strings.Contains(timeString, "month"):
		dur = time.Since(time.Now().AddDate(0, -1, 0))
	case strings.Contains(timeString, "year"):
		dur = time.Since(time.Now().AddDate(-1, 0, 0))
	default:
		dur = time.Since(time.Now().AddDate(0, 0, -7))
	}

	r := regexp.MustCompile("[0-9]+")
	mult, _ := strconv.Atoi(r.FindString(timeString))
	if mult < 1 {
		mult = 1
	}

	return dur * time.Duration(mult)
}

func Contains(s []string, e string) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}

	return false
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
