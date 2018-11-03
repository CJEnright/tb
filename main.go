package main

import (
	"time"
	"strings"
	"fmt"
	"flag"
	"sort"
	"encoding/json"
	"os"
	"io/ioutil"

	clr "github.com/logrusorgru/aurora"
)

type Project struct {
	Name string `json:"name"`
	Active bool `json:"active"`
	StartedAt time.Time `json:"startTime"`
	Duration time.Duration `json:"duration"`
}

var (
	projects []Project
)

func main() {
	load("./tb.json")

	var err error
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
		} else if os.Args[1] == "-a" {
			stats()
		}
	}

	save("./tb.json")

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
	flagAll := flag.Bool("a", false, "show timers not currently running")
	flag.Parse()

	if len(os.Args) == 2 {
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

					name = name[strings.LastIndex(name, "/") + 1:len(name)]
				}
			}


			if p.Active {
				fmt.Printf(
					"%s%s is %s with a total duration of %s\n",
					spacing,
					clr.Bold(name),
					clr.Green("running"),
					p.Duration.Truncate(time.Second).String(),
				)
			} else if *flagAll {
				fmt.Printf(
					"%s%s is %s with a total duration of %s\n",
					spacing,
					clr.Bold(name),
					"not running",
					p.Duration.Truncate(time.Second).String(),
				)
			}
		}
	} else if len(os.Args) == 3 {
		if os.Args[2] == "week" {

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

		return err
	} else {
		projectName = os.Args[2]

		if IndexOf(projects, projectName) == -1 {
			p := Project{Name: projectName}
			projects = append(projects, p)
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
			projects[index].Active = true
			projects[index].StartedAt = time.Now()

			fmt.Printf("Started \"%s\" at %s\n", projectName, projects[index].StartedAt.String())
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
			projects[index].Active = false
			projects[index].Duration += time.Now().Sub(projects[index].StartedAt)

			fmt.Printf("Stopped \"%s\" at a duration of %s\n", projectName, projects[index].Duration.Truncate(time.Second).String())
		} else {
			return fmt.Errorf("Project \"%s\" isn't running", projectName)
		}
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
