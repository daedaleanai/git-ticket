## git-ticket user

Display or change the user identity.

### Synopsis

Display or change the user identity.

```
git-ticket user [<user name/id>] [flags]
```

### Options

```
  -f, --field string   Select field to display. Valid values are [email,humanId,id,keys,lastModification,lastModificationLamport,login,metadata,name,phabId]
  -h, --help           help for user
```

### Options inherited from parent commands

```
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket](git-ticket.md)	 - A ticket tracker embedded in Git.
* [git-ticket user adopt](git-ticket_user_adopt.md)	 - Adopt an existing identity as your own.
* [git-ticket user create](git-ticket_user_create.md)	 - Create a new identity.
* [git-ticket user edit](git-ticket_user_edit.md)	 - Edit a user identity.
* [git-ticket user key](git-ticket_user_key.md)	 - Display, add or remove keys to/from a user.
* [git-ticket user ls](git-ticket_user_ls.md)	 - List identities.

