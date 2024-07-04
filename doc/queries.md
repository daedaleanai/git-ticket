# Searching bugs

You can search bugs using a query language for both filtering, sorting and coloring (on the webui). A query could look like this:

```
status(open) sort(edit) color-by(ccb("John"))
```

A few tips:

- queries are case insensitive, except when using regular expressions, in which case they are case sensitive/insensitive based on the given regular expression.
- you can combine as many qualifiers as you want using grouping expressions like `all(...)` and `any(...)`.
- filter expressions can be nested: `all(any(label(a), label(b)), label(r"workflow"))`.
- filter expressions can contain regular expressions of the form `r"..."`.
- you can use double quotes for multi-word search terms. For example, `author("René Descartes")` searches for bugs opened by René Descartes.
- instead of a complete ID, you can use any prefix length (except 0). For example `participant(9ed1a)`. 

## Literal matcher types

The following types of literal matchers are supported:

| Literal matchers | Description                                                                                       | Example                                                 |
| ---              | ---                                                                                               | ---                                                     |
| `Identifier`     | Text identifier that cannot contain parenthesis, commas, double-quotes or whitespaces.            | `identifier:example`                                    |
| `String`         | Text string delimited by double-quotes. Cannot contain double-quotes.                             | `"identifier:example with spaces and parenthesis()"`    |
| `Regexp`         | Regular expresion delimited by double-quotes and precided by an `r`. Cannot contain double-quotes.| `r"repo:.*"`                                            |

## Filtering

The following filters are available:

| Filter nodes     | Arguments                                                            | Example                                                                                               |
| ---              | ---                                                                  | ---                                                                                                   |
| `status`         | Comma separated list of statuses. May be surrounded in double-quotes | `status(proposed, vetted)` matches tickets in either the proposed or vetted status                    |
| `author`         | A literal matcher                                                    | `author(r"John|Jane")` matches tickets authored by either John or Jane                                |
| `assignee`       | A literal matcher                                                    | `assignee(r"John|Jane")` matches tickets assigned to either John or Jane                              |
| `ccb`            | A literal matcher                                                    | `ccb(john)` matches tickets CCB'ed by John                                                            |
| `ccb-pending`    | A literal matcher                                                    | `ccb-pending(john)` matches tickets CCB'ed by John in which a CCB action is pending                   |
| `actor`          | A literal matcher                                                    | `actor(r"John|Jane")` matches tickets in which either John or Jane are actors                         |
| `participant`    | A literal matcher                                                    | `participant(r"John|Jane")` matches tickets in which either John or Jane are participants             |
| `label`          | A literal matcher                                                    | `label(r"^repo:.*")` matches tickets with labels that start with `repo:`                              |
| `title`          | A literal matcher                                                    | `title(r"^\[QA\].*")` matches tickets in which their title starts with `[QA]`                         |
| `not`            | A nested filter                                                      | `not(title(r"^\[QA\].*"))` matches tickets that do not have titles starting with `[QA]`               |
| `any`            | A comma-separated list of nested filters                             | `any(ccb(john), status(vetted))` matches tickets that are CCB'ed by John or are in the vetted status  |
| `all`            | A comma-separated list of nested filters                             | `all(ccb(john), status(vetted))` matches tickets that are CCB'ed by John and are in the vetted status |
| `created-before` | Identifier or string with format 2006-01-02T15:04:05 or 2006-01-02   | `created-before(2006-01-02)` matches tickets created before the given date                            |
| `created-after`  | Identifier or string with format 2006-01-02T15:04:05 or 2006-01-02   | `created-after(2006-01-02)` matches tickets created before the given date                             |
| `edit-before`    | Identifier or string with format 2006-01-02T15:04:05 or 2006-01-02   | `edit-before(2006-01-02)` matches tickets were last edited before the given date                      |
| `edit-after`     | Identifier or string with format 2006-01-02T15:04:05 or 2006-01-02   | `edit-after(2006-01-02)` matches tickets were last edited after the given date                        |

## Sorting

You can sort results by adding a `sort()` expression to your query. “Descending” means most recent time or largest ID first, whereas “Ascending” means oldest time or smallest ID first.

Note: to deal with differently-set clocks on distributed computers, `git-ticket` uses a logical clock internally rather than timestamps to order bug changes over time. That means that the timestamps recorded might not match the returned ordering. More on that in [the documentation](model.md#you-cant-rely-on-the-time-provided-by-other-people-their-clock-might-by-off-for-anything-other-than-just-display)

### Sort by Id

| Sort nodes                   | Example                                     |
| ---                          | ---                                         |
| `sort(id-desc)`              | will sort bugs by their descending Ids      |
| `sort(id)` or `sort(id-asc)` | will sort bugs by their ascending Ids       |

### Sort by Creation time

You can sort bugs by their creation time.

| Sort nodes                   | Example                                     |
| ---                                       | ---                                              |
| `sort(creation)` or `sort(creation-desc)` | will sort bugs by their descending creation time |
| `sort(creation-asc)`                      | will sort bugs by their ascending creation time  |

### Sort by Edit time

You can sort bugs by their edit time.

| Sort nodes                        | Example                                              |
| ---                               | ---                                                  |
| `sort(edit)` or `sort(edit-desc)` | will sort bugs by their descending last edition time |
| `sort(edit-asc)`                  | will sort bugs by their ascending last edition time  |

## Coloring

The webui can color tickets that match a certain criteria. All coloring nodes start with `color-by()` and contain a single argument, which must be one of:

| Color-by nodes   | Arguments                                                            | Example                                                                                                                                           |
| ---              | ---                                                                  | ---                                                                                                                                               |
| `author`         | A literal matcher                                                    | `color-by(author(r"John|Jane"))` matches tickets authored by either John or Jane, coloring the tickets that were authored by them.                |
| `assignee`       | A literal matcher                                                    | `color-by(assignee(r"John|Jane"))` matches tickets assigned to either John or Jane, coloring the tickets that are assigned to them.               |
| `ccb-pending`    | A literal matcher                                                    | `color-by(ccb-pending(john))` matches tickets CCB'ed by John in which a CCB action is pending, coloring the tickets that are pending CCB by John. |
| `label`          | A literal matcher                                                    | `color-by(label(r"^repo:.*"))` matches tickets with labels that start with `repo:`, assigning a color to each of the different matched labels.    |
