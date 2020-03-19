# tb
Bit of a time tracking meme

```
tb - time tracking

commands:
	tb               Which projects are running
	tb new name      Register a new project

	tb start name    Start tracking a project (can also match by suffix)
	tb stop name     Stop tracking a project (can also match by suffix)
	tb s name        Toggle on/off tracking a project

	tb archive name  Archive a project so it's not seen any more
	tb recover name  Recover a project so it's not archived any more

	tb stats [time]  How long each non-archived project has been running
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
	Everything tb needs is stored in one JSON file at ~/.tb.json
```
