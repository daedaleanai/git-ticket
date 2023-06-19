## git-ticket

A ticket tracker embedded in Git.

### Synopsis

git-ticket is a ticket tracker embedded in git.

git-ticket uses git objects to store the tickets history separate from the main repo
history. As tickets are regular git objects, they can be pushed and pulled from/to
the same git remote you are already using to collaborate with other people.



```
git-ticket [flags]
```

### Options

```
  -h, --help            help for git-ticket
      --rebuild-cache   force the cache to be rebuilt
```

### SEE ALSO

* [git-ticket add](git-ticket_add.md)	 - Create a new ticket.
* [git-ticket assign](git-ticket_assign.md)	 - Assign a user to a ticket.
* [git-ticket ccb](git-ticket_ccb.md)	 - Change Control Board (CCB) actions of a ticket.
* [git-ticket commands](git-ticket_commands.md)	 - Display available commands.
* [git-ticket comment](git-ticket_comment.md)	 - Display or add comments to a ticket.
* [git-ticket config](git-ticket_config.md)	 - List configs or show the specified config
* [git-ticket deselect](git-ticket_deselect.md)	 - Clear the implicitly selected ticket.
* [git-ticket label](git-ticket_label.md)	 - Display, add or remove labels to/from a ticket.
* [git-ticket ls](git-ticket_ls.md)	 - List tickets.
* [git-ticket ls-id](git-ticket_ls-id.md)	 - List ticket identifiers.
* [git-ticket ls-label](git-ticket_ls-label.md)	 - List valid labels.
* [git-ticket pull](git-ticket_pull.md)	 - Pull tickets update from a git remote.
* [git-ticket push](git-ticket_push.md)	 - Push tickets update to a git remote.
* [git-ticket review](git-ticket_review.md)	 - Review actions of a ticket.
* [git-ticket rm](git-ticket_rm.md)	 - Remove an existing ticket.
* [git-ticket select](git-ticket_select.md)	 - Select a ticket for implicit use in future commands.
* [git-ticket show](git-ticket_show.md)	 - Display the details of a ticket.
* [git-ticket status](git-ticket_status.md)	 - Display or change a ticket status.
* [git-ticket termui](git-ticket_termui.md)	 - Launch the terminal UI.
* [git-ticket title](git-ticket_title.md)	 - Display or change a title of a ticket.
* [git-ticket user](git-ticket_user.md)	 - Display or change the user identity.
* [git-ticket validate](git-ticket_validate.md)	 - Validate identities and commits signatures.
* [git-ticket version](git-ticket_version.md)	 - Show git-ticket version information.

