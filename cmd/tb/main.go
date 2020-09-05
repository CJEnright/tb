package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cjenright/tb"
)

// TODO log command - show entries in chronological order
// ^^ yikes that'll be expensive

const (
	helpText = `tb - time tracking

commands:
	tb               Which projects are running
	tb new name      Register a new project

	tb start name [note]  Start tracking a project (can also match by suffix)
	tb stop name [note]   Stop tracking a project (can also match by suffix)
	tb s name [note]      Toggle on/off tracking a project

	tb archive name  Archive a project so it's not seen any more
	tb recover name  Recover a project so it's not archived any more

	tb stats [time]          How long each non-archived project has been running
	tb timecard name [time]  Print a timecard for a project

	tb help          Show this help page

names:
	Projects can be arranged hierarchically using "/".  For example, you could
	have a parent project called "school" and create a child project using:

		tb new school/cs193

	The full path name for that new project is "school/cs193", and when getting
	the stats for the school project tb will also count time tracked for the
	"school/cs193" project.

suffix matching:
	Having to type out the full path name of projects is annoying, so you can
	also match by suffix.  In the example above if there were only two projects,
	"school" and "school/cs193", you could start "school/cs193" with any of these
	commands:
	  
		tb start 3
		tb start 93
		tb start 193
		...

	If there are multiple projects with a matching suffix tb will prompt you to
	choose one.

times:
	Times are given in units since today, these are the units recognized and
	their abbreviations:

		Hours  - hour,  hr, h
		Days   - day,   dy, d
		Weeks  - week,  wk, w
		Months - month, mo, m 
		Years  - year,  yr, y 

	The way times are parsed is pretty flexible, so any of these would be valid
	time strings:
	  
		1w3d5y
		4 weeks
		14 weeks 2 hours
		...

storage:
	Everything tb needs is stored in one JSON file at ~/.tb.json `
)

func main() {
	path := os.Getenv("HOME") + "/.tb.json"
	tbw, err := tb.Load(path)
	if err != nil {
		fmt.Println("failed to load:", err.Error())
		return
	}

	didEdit := false

	l := len(os.Args)
	if l == 1 {
		tbw.Status()
	} else if l == 2 {
		command := strings.ToLower(os.Args[1])

		switch command {
		case "stats":
			tbw.Stats()
		case "help":
			fmt.Println(helpText)
		case "recalc", "recalculate":
			tbw.Recalculate()
			didEdit = true
		default:
			fmt.Printf("tb: unknown command %v\n", command)
		}
	} else {
		command := strings.ToLower(os.Args[1])
		projectName := os.Args[2]

		switch command {
		case "new":
			err := tbw.New(projectName)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			didEdit = true
		case "start":
			p := getProject(tbw, projectName)
			err = p.Start()
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			didEdit = true
		case "stop":
			p := getProject(tbw, projectName)
			err = p.Stop()
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			didEdit = true
		case "s":
			p := getProject(tbw, projectName)
			if p.IsRunning == true {
				err = p.Stop()
			} else {
				err = p.Start()
			}
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			didEdit = true
		case "timecard":
			p := getProject(tbw, projectName)
			p.Timecard(*tbw.Conf)
		case "archive":
			p := getProject(tbw, projectName)
			p.Archive()
			didEdit = true
		case "stats":
			tbw.Stats()
		default:
			fmt.Printf("tb: unknown command %v\n", command)
		}
	}

	if didEdit {
		err = tbw.Save(path)
		if err != nil {
			fmt.Println("failed to save:", err.Error())
			return
		}
	}
}

func getProject(tbw *tb.TBWrapper, name string) *tb.Project {
	matches := tbw.Root.FindProjects("/" + name)

	if len(matches) > 1 {
		selection := 0

		for selection < 1 || selection > len(matches) {
			fmt.Printf("multiple projects found with suffix \"%s\":\n", name)

			for i, v := range matches {
				fmt.Printf("(%d) %s\n", i+1, v.Name)
			}

			_, err := fmt.Scanln(&selection)
			if err != nil {
				fmt.Printf("unable to handle input selection: %v\n", err)
				os.Exit(1)
			}
		}

		return matches[selection-1]
	} else if len(matches) == 1 {
		return matches[0]
	} else {
		fmt.Printf("no project matching name %s was found\n", name)
		os.Exit(1)
		return nil
	}
}
