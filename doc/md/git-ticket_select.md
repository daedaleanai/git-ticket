## git-ticket select

Select a ticket for implicit use in future commands.

### Synopsis

Select a ticket for implicit use in future commands.

This command allows you to omit any ticket ID argument, for example:
  git ticket show
instead of
  git ticket show 2f153ca

The complementary command is "git ticket deselect" performing the opposite operation.


```
git-ticket select ID [flags]
```

### Examples

```
git ticket select 2f15
git ticket comment
git ticket status

```

### Options

```
  -h, --help   help for select
```

### Options inherited from parent commands

```
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket](git-ticket.md)	 - A ticket tracker embedded in Git.

