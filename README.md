# tb
Bit of a time tracking meme

## Usage
`tb` - gives a list of running timers  
`tb -a` - gives a list all timers  
`tb -A` - gives a list all timers including archived ones  
`tb new projname` - creates a new project  
`tb new projname/subproj` - creates a new subproject  
`tb start projname` - starts tracking project  
`tb stop projname` - stops tracking project  
`tb stats day` - show time tracked for the past day (also accepts week and month)  
`tb archive projname` - archive a project  

## Installation
1. Clone this repo
2. Have Go installed
3. Inside the repo, `go install`

## Other stuff
All projects and segments are stored in `~/.tb.json`, should be pretty easy to edit or whatever you want to do with it.

Projects can be nested with a slash, so `school/classname` will be a subproject of school, and all time tracked specifically to `classname` will be added to the total time shown by `school`

### Example
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
> logged school for 10m30s in the past day
> logged school/cs250 for 7m25s in the past day
> logged school/engl106 for 3m5s in the past day
```

lol idk if that makes sense but it works for me so you do you
