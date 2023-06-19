## git-ticket review fetch

Get Differential Revision data from Phabricator and store in a ticket.

### Synopsis

fetch stores Phabricator Differential Revision data in a ticket.

The command takes a Phabricator Differential Revision ID (e.g. D1234) and queries the
Phabricator server for any associated comments or status changes, any resulting data
is stored with the selected ticket. Subsequent calls with the same ID will fetch and
store any updates since the previous call. Multiple Revisions can be stored with a
ticket by running the command with different IDs.



```
git-ticket review fetch DIFF-ID [ID] [flags]
```

### Options

```
  -h, --help   help for fetch
```

### Options inherited from parent commands

```
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket review](git-ticket_review.md)	 - Review actions of a ticket.

