# tb
Bit of a time tracking meme

## Commands
`stats` - stats for all projects in a time period
`new` - create a new project
`start` - start tracking time
`stop` - stop time tracking
`s` - toggle start/stop a project
`timecard` - get a timecard printout of time segments
`archive` - don't show this project in printouts

## In practice
`tb` or `tb status` - gives a list of running timers  
`tb new projname` - creates a new project  
`tb new projname/sub` - creates a new subproject  
`tb start projname` - starts tracking project  
`tb start sub` - start a project based off of suffix
`tb stop projname` - stops tracking project  
`tb stats 2 days` - show time tracked for the past 2 days (flexible, accepts hour/day/week/month/year)  
`tb timecard projname 2 days` - similar to stats but 
`tb archive projname` - archive a project  

## Installation
1. Clone this repo
2. Have Go installed
3. Inside the repo, run `$ go install`

## Other stuff
All projects and segments are stored in `~/.tb.json`, should be pretty easy to edit or whatever you want to do with it.

Projects can be nested with a slash, so `school/classname` will be a subproject of school, and all time tracked specifically to `classname` will be added to the total time shown by `school`

## Example
```
$ tb new school
$ tb new school/cs250
$ tb new school/engl106
$ tb start school/cs250
... time passes ...
$ tb stop school/cs250
$ tb start school/engl106
... time passes ...
$ tb stop school/engl106
$ tb stats day
school for 38m16s in the past day
↳ cs250 for 18m13s in the past day
↳ engl106 for 20m3s in the past day
$ tb timecard school/cs250
Project        Date    Start      End        Duration
school/cs250   01/16   09:14:00   09:32:13   18m13s
-----------------------------------------------------
Total duration: 18m13s in the past week
$ tb timecard school
Project          Date    Start      End        Duration
school/cs250     01/16   09:14:00   09:32:13   18m13s
school/engl106   01/16   00:14:21   00:34:24   20m3s
-----------------------------------------------------
Total duration: 38m16s in the past week
```

lol idk if that makes sense but it works for me so you do you
