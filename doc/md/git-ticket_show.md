## git-ticket show

Display the details of a ticket.

### Synopsis

Display the details of a ticket.

```
git-ticket show [<ticket id>] [flags]
```

### Options

```
  -t, --timeline        Output the timeline of the ticket
      --field string    Select field to display. Valid values are [assignee,author,authorEmail,ccb,checklists,createTime,lastEdit,humanId,id,labels,reviews,shortId,status,nextStatuses,title,workflow,actors,participants]
  -f, --format string   Select the output formatting style. Valid values are [default,json,org-mode] (default "default")
  -s, --since string    Limit the timeline to changes since the given date/time. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd
  -h, --help            help for show
```

### Options inherited from parent commands

```
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket](git-ticket.md)	 - A ticket tracker embedded in Git.

