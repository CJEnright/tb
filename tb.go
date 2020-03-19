package tb

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

const (
	DefaultDateFormat = "1/31"     // Month/Day
	DefaultTimeFormat = "15:04:05" // HH:MM:SS, 24 hr
)

// A Config defines how date and times are formatted
type Config struct {
	DateFormat string `json:"date_format"`
	TimeFormat string `json:"time_format"`
}

// A tbWrapper wraps a Config and a list of projects
type tbWrapper struct {
	Conf     Config     `json:"config"`
	Projects []*Project `json:"projects"`
}

func (tb *tbWrapper) New(name string) error {
	p, err := tb.FindProject(name)
	if err == nil && p.Name == name {
		fmt.Printf("project with name \"%s\" already exists\n", p.Name)
		return err
	} else {
		// Add base projects so stats come out right
		slashes := strings.Split(name, "/")
		for i := 0; i < len(slashes)-1; i++ {
			combined := strings.Join(slashes[:i+1], "/")
			// Make sure project with this name doesn't exist
			_, err = tb.FindProject(combined)
			if err != nil {
				p := Project{Name: combined}
				tb.Projects = append(tb.Projects, &p)
			}
		}

		p := Project{Name: name}
		tb.Projects = append(tb.Projects, &p)

		fmt.Printf("created project \"%s\"\n", p.Name)
		err = nil
	}

	return err
}

// Status shows which projects are currently running.
func (tb *tbWrapper) Status() {
	for _, p := range tb.Projects {
		p.Status()
	}
}

// Stats prints the amount of time logged for each project.
func (tb *tbWrapper) Stats() {
	var dur time.Duration
	durString := "week"
	if len(os.Args) < 3 {
		dur = time.Since(time.Now().AddDate(0, 0, -7))
	} else {
		dur, durString = parseTimeString(2)
	}

	fmt.Printf("Stats for the past %s:\n", durString)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, p := range tb.Projects {
		p.Stats(w, "", dur, durString)
	}
	w.Flush()
}

// FindProject finds a project that either has the prefix name or has
// a full path name equal to the name parameter.
func (tb *tbWrapper) FindProject(name string) (*Project, error) {
	matches := []*Project{}
	for _, p := range tb.Projects {
		matches = append(matches, p.findProject(name)...)
	}

	if len(matches) > 1 {
		selection := 0

		for selection < 1 || selection > len(matches) {
			fmt.Printf("multiple projects found with suffix \"%s\":\n", name)

			for i, v := range matches {
				fmt.Printf("(%d) %s\n", i+1, v.Name)
			}

			_, err := fmt.Scanln(&selection)
			if err != nil {
				return nil, err
			}
		}

		return matches[selection-1], nil
	} else if len(matches) == 1 {
		return matches[0], nil
	} else {
		return nil, ErrProjectNotFound
	}
}

// parseTimeString finds a duration from strings like
// "1w", "1y3d", "week", "1 month", etc.
// It returns the duration that string represents and a cleaned
// non abbreviated version of that string
func parseTimeString(argStartIndex int) (dur time.Duration, cleanString string) {
	if len(os.Args) <= argStartIndex {
		return time.Since(time.Now().AddDate(0, 0, -7)), "week"
	}

	timeString := strings.ToLower(strings.Join(os.Args[argStartIndex:], " "))
	hasNumbers := strings.ContainsAny(timeString, "0123456789")

	if !hasNumbers {
		// No multipliers, can assume that there's just one
		// unit of time being asked for.
		dur, cleanString = abbrvToDuration(timeString)
	} else {
		// Assumes all units of time are preceded by a number
		i := 0

		for i < len(timeString) {
			// Consume any characters that might come before the number
			for i < len(timeString) && !strings.Contains("0123456789", string(timeString[i])) {
				i++
			}

			// Consume the number
			num := ""
			for i < len(timeString) && strings.Contains("0123456789", string(timeString[i])) {
				num += string(timeString[i])
				i++
			}

			// Consume up until the next number
			newTimeString := ""
			for i < len(timeString) && !strings.Contains("0123456789", string(timeString[i])) {
				newTimeString += string(timeString[i])
				i++
			}

			multiplier, _ := strconv.Atoi(num)
			newDur, durStr := abbrvToDuration(newTimeString)
			durStr = num + " " + durStr

			if multiplier > 1 {
				durStr += "s"
			}

			dur += newDur * time.Duration(multiplier)
			cleanString += durStr + " "
		}
	}

	return dur, cleanString
}

// Check if a string contains any of the strings in an array
func containsAny(e string, s []string) bool {
	for _, v := range s {
		if strings.Contains(e, v) {
			return true
		}
	}

	return false
}

var (
	hourAbbrvs  = []string{"hour", "hr", "h"}
	dayAbbrvs   = []string{"day", "dy", "d"}
	weekAbbrvs  = []string{"week", "wk", "w"}
	monthAbbrvs = []string{"month", "mo", "m"}
	yearAbbrvs  = []string{"year", "yr", "y"}
)

// abbrvToDuration converts what might be an abbreviated unit of time
// (like hr or h) to a known unit of time.
func abbrvToDuration(input string) (dur time.Duration, s string) {
	switch {
	case containsAny(input, hourAbbrvs):
		dur = time.Since(time.Now().Add(-1 * time.Hour))
		s = "hour"
	case containsAny(input, dayAbbrvs):
		dur = time.Since(time.Now().AddDate(0, 0, -1))
		s = "day"
	case containsAny(input, weekAbbrvs):
		dur = time.Since(time.Now().AddDate(0, 0, -7))
		s = "week"
	case containsAny(input, monthAbbrvs):
		dur = time.Since(time.Now().AddDate(0, -1, 0))
		s = "month"
	case containsAny(input, yearAbbrvs):
		dur = time.Since(time.Now().AddDate(-1, 0, 0))
		s = "year"
	default:
		// Default to one week if we can't figure anything out
		dur = time.Since(time.Now().AddDate(0, 0, -7))
		s = "week"
	}

	return dur, s
}

// Load loads a file for editing time tracking information.
func Load(path string) (tb *tbWrapper, err error) {
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

// Save writes a tb file for long term storage.
func (tb *tbWrapper) Save(path string) (err error) {
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
