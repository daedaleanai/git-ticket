## git-ticket user create

Create a new identity.

### Synopsis

Create a new identity.

```
git-ticket user create [flags]
```

### Options

```
      --key-file string   Take the armored PGP public key from the given file. Use - to read the message from the standard input
  -s, --skipPhabId        Do not attempt to retrieve the users Phabricator ID (note: fetching reviews where they commented will fail if it is not set)
  -h, --help              help for create
```

### Options inherited from parent commands

```
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket user](git-ticket_user.md)	 - Display or change the user identity.

