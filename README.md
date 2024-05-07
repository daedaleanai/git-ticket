<p align="center">
    <img width="150px" src="https://cdn.rawgit.com/MichaelMure/git-bug/master/misc/logo/logo-alpha-flat-bg.svg">
</p>
<h1 align="center">git-ticket</h1>

<div align="center">

[![Build Status](https://travis-ci.org/daedaleanai/git-ticket.svg?branch=master)](https://travis-ci.org/daedaleanai/git-ticket)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3+-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)
[![GoDoc](https://godoc.org/github.com/daedaleanai/git-ticket?status.svg)](https://godoc.org/github.com/daedaleanai/git-ticket)
[![Go Report Card](https://goreportcard.com/badge/github.com/daedaleanai/git-ticket)](https://goreportcard.com/report/github.com/daedaleanai/git-ticket)

</div>

`git-ticket` is a bug tracker that:

- **is fully embedded in git**: you only need your git repository to have a bug tracker
- **is distributed**: use your normal git remote to collaborate, push and pull your bugs!
- **works offline**: in a plane or under the sea? Keep reading and writing bugs!
- **prevents vendor lock-in**: your usual service is down or went bad? You already have a full backup.
- **is fast**: listing bugs or opening them is a matter of milliseconds
- **doesn't pollute your project**: no files are added in your project
- **integrates with your tooling**: use the UI you like (CLI, terminal, web) or integrate with your existing tools through the CLI or the GraphQL API
- **bridges to other bug trackers**: use [bridges](#bridges) to import and export to other trackers.

:construction: This is now more than a proof of concept, but still not fully stable. Expect dragons and unfinished business. :construction:

## Installation

```shell
git clone git@github.com:daedaleanai/git-ticket.git
make install
```

If it's not done already, add the golang binary directory in your PATH:

```bash
export PATH=$PATH:$(go env GOROOT)/bin:$(go env GOPATH)/bin
```


## Workflows

There are multiple ways to use `git-bug`:

<details><summary>Native workflow</summary>
<p align="center">
    <img src="misc/diagrams/native_workflow.png" alt="Native workflow">
</p>

This is the pure `git-bug` experience. In a similar fashion as with code, use `git bug push` and `git bug pull` to push and pull your bugs between git remotes and collaborate with your teammate. 

</details>

<details><summary>Bridge workflow</summary>
<p align="center">
    <img src="misc/diagrams/bridge_workflow.png" alt="Bridge workflow">
</p>

As `git-bug` has bridges with other bug-trackers, you can use it as your personal local remote interface. Sync with `git bug bridge pull` and `git bug bridge push`, work from your terminal, integrate into your editor, it's up to you. And it works offline !

</details>

<details><summary>Web UI workflow (WIP)</summary>
<p align="center">
    <img src="misc/diagrams/webui-workflow.png" alt="Web UI workflow">
</p>

Often, projects needs to have their bug-tracker public and accept editions from anyone facing a problem. To support this workflow, `git-bug` aims to have the web UI accept external OAuth authentication and act as a public portal. However the web UI is not up to speed for that yet. Contribution are very much welcome!

</details>

## CLI usage

Create a new identity:

```
git ticket user create
```

Create a new ticket:

```
git ticket add
```

Your favorite editor will open to write a title and a message.

You can push your new entry to a remote:
```
git ticket push [<remote>]
```

And pull for updates:
```
git ticket pull [<remote>]
```

List existing tickets:
```
git ticket ls
```

Filter and sort tickets using a [query](doc/queries.md):
```
git ticket ls "status:open sort:edit"
```

You can now use commands like `show`, `comment`, `open` or `close` to display and modify tickets. For more details about each command, you can run `git ticket <command> --help` or read the [command's documentation](doc/md/git-ticket.md).

## Interactive terminal UI

An interactive terminal UI is available using the command `git ticket termui` to browse and edit tickets.

![Termui recording](misc/termui_recording.gif)

<!--
## Web UI (status: WIP)

You can launch a rich Web UI with `git ticket webui`.

<p align="center">
  <img src="misc/webui1.png" alt="Web UI screenshot 1" width="880">
</p>

<p align="center">
  <img src="misc/webui2.png" alt="Web UI screenshot 2" width="880">
</p>

This web UI is entirely packed inside the same go binary and serve static content through a localhost http server.

The web UI interact with the backend through a GraphQL API. The schema is available [here](api/graphql/schema).

## Bridges

### Importer implementations

|                                                 | Github             | Gitlab             | Jira               | Launchpad          |
|-------------------------------------------------|--------------------|--------------------|--------------------|--------------------|
| **incremental**<br/>(can import more than once) | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| **with resume**<br/>(download only new data)    | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| **identities**                                  | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: |
| identities update                               | :x:                | :x:                | :x:                | :x:                |
| **bug**                                         | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: |
| comments                                        | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: |
| comment editions                                | :heavy_check_mark: | :x:                | :heavy_check_mark: | :x:                |
| labels                                          | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| status                                          | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| title edition                                   | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| **media/files**                                 | :x:                | :x:                | :x:                | :x:                |
| **automated test suite**                        | :heavy_check_mark: | :heavy_check_mark: | :x:                | :x:                |

### Exporter implementations

|                          | Github             | Gitlab             | Jira               | Launchpad |
|--------------------------|--------------------|--------------------|--------------------|-----------|
| **bug**                  | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| comments                 | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| comment editions         | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| labels                   | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| status                   | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| title edition            | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| **automated test suite** | :heavy_check_mark: | :heavy_check_mark: | :x:                | :x:       |

#### Bridge usage

Interactively configure a new github bridge:

```bash
git ticket bridge configure
```

Or manually:

```bash
git ticket bridge configure \
    --name=<bridge> \
    --target=github \
    --url=https://github.com/daedaleanai/git-ticket \
    --login=<login>
    --token=<token>
```

Import bugs:

```bash
git ticket bridge pull [<name>]
```

Export modifications:

```bash
git ticket bridge push [<name>]
```

Deleting a bridge:

```bash
git ticket bridge rm [<name>]
```
-->

## Web UI

A simple experimental read-only web UI is available using the command `git ticket webui` to browse tickets on a kanban-style board. git ticket will expose the web UI on a local port (defaults to 3333, but can be changed using the `--port` parameter).

### Filtering and Sorting

Tickets shown in the web UI can filtered and sorted by providing a query as the `q` URL parameter.
See [here](doc/queries.md) for a documentation of the query language.

### Coloring

In addition, the web UI also supports a `color-by` qualifier in queries to color tickets based on author, assignee or labels.

| Qualifier                    | Example                                                    |
| ---                          | ---                                                        |
| `color-by:author`            | colors tickets by author                                   |
| `color-by:assignee`          | colors tickets by assignee                                 |
| `color-by:label:some-prefix` | colors tickets based on labels starting with `some-prefix` |

### Bookmarks

Queries can be bookmarked by storing a JSON array with the following structure in the local git config key `git-bug.webui.bookmarks`.
Bookmarks are organized in named groups, with each bookmark having a label and an associated [query](doc/queries.md) property.

```
[
    {
        "group": "me",
        "bookmarks": [
            {
                "label": "created",
                "query": "author:me"
            },
            {
                "label": "assigned",
                "query": "assignee:me"
            }
        ]
    },
    {
        "group": "milestones",
        "bookmarks": [
            {
                "label": "milestone 1",
                "query": "label:milestone-1"
            },
            {
                "label": "milestone 2",
                "query": "label:milestone-2"
            }
        ]
    }
]
```

Note: The git config value may not contain any newlines. The easiest way to add configure bookmarks is to edit the `.git/config` file directly rather than using the `git` command. E.g. to add the bookmarks from the above example, add the following lines to the `.git/config` file.

```
[git-bug "webui"]
	bookmarks = "[{\"group\":\"me\",\"bookmarks\":[{\"label\":\"created\",\"query\":\"author:me\"},{\"label\":\"assigned\",\"query\":\"assignee:me\"}]},{\"group\":\"milestones\",\"bookmarks\":[{\"label\":\"milestone1\",\"query\":\"label:milestone-1\"},{\"label\":\"milestone2\",\"query\":\"label:milestone-2\"}]}]"
```

### Cross-References

The detailed ticket view supports rendering certain patterns in the ticket's title, description or comments as clickable links.
This behavior is controlled by a set of rules, which can be configured via the git ticket config key `webui`.
The configuration is a JSON object with a single key `xref` that contains a array of rule objects.
Each rule consists of a regular expression (pattern) and a link template. Each match of a pattern is transformed to a clickable link.
The link target is constructed by instantiating the link template associated with the pattern. The Nth capture groups of the regular expression can be referenced in the link template by using `{{ index . N}}` notation. Capture group 0 always refers to the full match.

The following examples create links for URLs and create links to the detailed ticked view of this web UI for ticket IDs, respectively, and can be set by running `git ticket config webui`.

```
{
  "xref": [
    {
      "pattern": "\\bhttps?://\\S+\\b",
      "link": "{{ index . 0 }}"
    },
    {
      "pattern": "\\b[a-f0-9]{7,64}\\b",
      "link": "/ticket?id={{ index . 0 }}"
    }
  ]
}
```

## Internals

Interested by how it works ? Have a look at the [data model](doc/model.md) and the [internal bird-view](doc/architecture.md).

## Misc

- [Bash completion](misc/bash_completion)
- [Zsh completion](misc/zsh_completion)
- [PowerShell completion](misc/powershell_completion)
- [ManPages](doc/man)

## Planned features

- media embedding
- more bridges
- extendable data model to support arbitrary bug tracker
- inflatable raptor

<!--
## Contribute

PRs accepted. Drop by the [Gitter lobby](https://gitter.im/the-git-bug/Lobby) for a chat or browse the issues to see what is worked on or discussed.

```shell
git clone git@github.com:daedaleanai/git-ticket.git
```

You can now run `make` to build the project, or `make install` to install the binary in `$GOPATH/bin/`.

To work on the web UI, have a look at [the dedicated Readme.](webui/Readme.md)


## Contributors :heart:

This project exists thanks to all the people who contribute.
<a href="https://github.com/daedaleanai/git-ticket/graphs/contributors"><img src="https://opencollective.com/git-bug/contributors.svg?width=890&button=false" /></a>


## Backers

Thank you to all our backers! 🙏 [[Become a backer](https://opencollective.com/git-bug#backer)]

<a href="https://opencollective.com/git-bug#backers" target="_blank"><img src="https://opencollective.com/git-bug/tiers/backer.svg?width=890"></a>


## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website. [[Become a sponsor](https://opencollective.com/git-bug#sponsor)]

<a href="https://opencollective.com/git-bug/sponsor/0/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/1/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/2/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/3/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/4/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/5/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/6/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/7/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/8/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/9/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/9/avatar.svg"></a>

-->

## License

Unless otherwise stated, this project is released under the [GPLv3](LICENSE) or later license © Michael Muré.

The git-bug logo by [Viktor Teplov](https://github.com/vandesign) is released under the [Creative Commons Attribution 4.0 International (CC BY 4.0)](misc/logo/LICENSE) license © Viktor Teplov.
