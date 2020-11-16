## git-ticket ls

List tickets.

### Synopsis

Display a summary of each ticket.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language or with flags.

```
git-ticket ls [QUERY] [flags]
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
  -s, --status strings        Filter by status. Valid values are [open,closed]
  -a, --author strings        Filter by author
  -p, --participant strings   Filter by participant
      --actor strings         Filter by actor
  -A, --assignee strings      Filter by assignee
  -l, --label strings         Filter by label
  -t, --title strings         Filter by title
  -n, --no strings            Filter by absence of something. Valid values are [label]
  -b, --by string             Sort the results by a characteristic. Valid values are [id,creation,edit] (default "creation")
  -d, --direction string      Select the sorting direction. Valid values are [asc,desc] (default "asc")
  -f, --format string         Select the output formatting style. Valid values are [default,plain,json,org-mode] (default "default")
  -h, --help                  help for ls
```

### SEE ALSO

* [git-ticket](git-ticket.md)	 - A ticket tracker embedded in Git.

