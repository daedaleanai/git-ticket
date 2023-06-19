## git-ticket ls

List tickets.

### Synopsis

Display a summary of each ticket. By default shows only "active" tickets, i.e. In Progress, In Review, Reviewed and Accepted.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language or with flags.

```
git-ticket ls [<query>] [flags]
```

### Examples

```
List vetted tickets sorted by last edition with a query:
git ticket ls status:vetted sort:edit-desc

List merged tickets sorted by creation with flags:
git ticket ls --status merged --by creation

```

### Options

```
  -s, --status strings         Filter by status. Valid values are [proposed,vetted,inprogress,inreview,reviewed,accepted,merged,done,rejected,ALL]
  -a, --author strings         Filter by author
  -A, --assignee strings       Filter by assignee
  -c, --ccb strings            Filter by ccb
  -p, --participant strings    Filter by participant
      --actor strings          Filter by actor
  -l, --label strings          Filter by label
  -t, --title strings          Filter by title
      --create-before string   Filter by created before. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd
      --create-after string    Filter by created after. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd
      --edit-before string     Filter by last edited before. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd
      --edit-after string      Filter by last edited after. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd
  -n, --no strings             Filter by absence of something. Valid values are [label]
  -b, --by string              Sort the results by a characteristic. Valid values are [id,creation,edit] (default "creation")
  -d, --direction string       Select the sorting direction. Valid values are [asc,desc] (default "asc")
  -f, --format string          Select the output formatting style. Valid values are [default,plain,json,org-mode] (default "default")
  -h, --help                   help for ls
```

### Options inherited from parent commands

```
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket](git-ticket.md)	 - A ticket tracker embedded in Git.

