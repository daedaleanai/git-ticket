package commands

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"time"

	termtext "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
	"github.com/daedaleanai/git-ticket/util/colors"
	"github.com/daedaleanai/git-ticket/util/text"
)

type lsOptions struct {
	query query.Query

	statusQuery       []string
	createBeforeQuery string
	createAfterQuery  string
	editBeforeQuery   string
	editAfterQuery    string
	noQuery           []string
	sortBy            string
	sortDirection     string
	outputFormat      string
}

func newLsCommand() *cobra.Command {
	env := newEnv()
	options := lsOptions{}

	cmd := &cobra.Command{
		Use:   "ls [query]",
		Short: "List tickets.",
		Long: `Display a summary of each ticket. By default shows only "active" tickets, i.e. In Progress, In Review, Reviewed and Accepted.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language or with flags.`,
		Example: `List vetted tickets sorted by last edition with a query:
git ticket ls status:vetted sort:edit-desc

List merged tickets sorted by creation with flags:
git ticket ls --status merged --by creation
`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLs(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringSliceVarP(&options.statusQuery, "status", "s", nil,
		"Filter by status. Valid values are [proposed,vetted,inprogress,inreview,reviewed,accepted,merged,done,rejected,ALL]")
	flags.StringSliceVarP(&options.query.Author, "author", "a", nil,
		"Filter by author")
	flags.StringSliceVarP(&options.query.Assignee, "assignee", "A", nil,
		"Filter by assignee, or specify UNASSIGNED")
	flags.StringSliceVarP(&options.query.Ccb, "ccb", "c", nil,
		"Filter by ccb members assigned to the ticket, or specify UNASSIGNED")
	flags.StringSliceVarP(&options.query.CcbPending, "ccb-pending", "C", nil,
		"Filter by ccb members assigned to the ticket, which are pending approval")
	flags.StringSliceVarP(&options.query.Participant, "participant", "p", nil,
		"Filter by participant")
	flags.StringSliceVarP(&options.query.Actor, "actor", "", nil,
		"Filter by actor")
	flags.StringSliceVarP(&options.query.Label, "label", "l", nil,
		"Filter by label")
	flags.StringSliceVarP(&options.query.Title, "title", "t", nil,
		"Filter by title")
	flags.StringVarP(&options.createBeforeQuery, "create-before", "", "",
		"Filter by created before. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd")
	flags.StringVarP(&options.createAfterQuery, "create-after", "", "",
		"Filter by created after. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd")
	flags.StringVarP(&options.editBeforeQuery, "edit-before", "", "",
		"Filter by last edited before. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd")
	flags.StringVarP(&options.editAfterQuery, "edit-after", "", "",
		"Filter by last edited after. Valid formats are: yyyy-mm-ddThh:mm:ss OR yyyy-mm-dd")
	flags.StringSliceVarP(&options.noQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	flags.StringVarP(&options.sortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	flags.StringVarP(&options.sortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")
	flags.StringVarP(&options.outputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,plain,json,org-mode]")

	return cmd
}

func runLs(env *Env, opts lsOptions, args []string) error {
	var q *query.Query
	var err error

	if len(args) >= 1 {
		// construct filter from query language
		q, err = query.Parse(strings.Join(args, " "))

		if err != nil {
			return err
		}
	} else {
		// construct filter from flags
		err = completeQuery(&opts)
		if err != nil {
			return err
		}
		q = &opts.query
	}
	if reflect.ValueOf(q.Filters).IsZero() {
		// If no filters are set then default to active tickets
		q.Status = bug.ActiveStatuses()
	} else if len(q.Status) == 0 {
		// If filters are set but not on status then apply to all tickets
		q.Status = bug.AllStatuses()
	}

	allIds := env.backend.QueryBugs(q)

	bugExcerpt := make([]*cache.BugExcerpt, len(allIds))
	for i, id := range allIds {
		b, err := env.backend.ResolveBugExcerpt(id)
		if err != nil {
			return err
		}
		bugExcerpt[i] = b
	}

	switch opts.outputFormat {
	case "org-mode":
		return lsOrgmodeFormatter(env, bugExcerpt)
	case "plain":
		return lsPlainFormatter(env, bugExcerpt)
	case "json":
		return lsJsonFormatter(env, bugExcerpt)
	case "default":
		return lsDefaultFormatter(env, bugExcerpt)
	default:
		return fmt.Errorf("unknown format %s", opts.outputFormat)
	}
}

type JSONBugExcerpt struct {
	Id         string   `json:"id"`
	HumanId    string   `json:"human_id"`
	CreateTime JSONTime `json:"create_time"`
	EditTime   JSONTime `json:"edit_time"`

	Status       string         `json:"status"`
	Labels       []bug.Label    `json:"labels"`
	Title        string         `json:"title"`
	Actors       []JSONIdentity `json:"actors"`
	Participants []JSONIdentity `json:"participants"`
	Author       JSONIdentity   `json:"author"`
	Assignee     JSONIdentity   `json:"assignee"`
	Ccb          []JSONCcbInfo  `json:"ccb"`

	Comments int               `json:"comments"`
	Metadata map[string]string `json:"metadata"`
}

func lsJsonFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	jsonBugs := make([]JSONBugExcerpt, len(bugExcerpts))
	for i, b := range bugExcerpts {
		jsonBug := JSONBugExcerpt{
			Id:         b.Id.String(),
			HumanId:    b.Id.Human(),
			CreateTime: NewJSONTime(b.CreateTime(), b.CreateLamportTime),
			EditTime:   NewJSONTime(b.EditTime(), b.EditLamportTime),
			Status:     b.Status.String(),
			Labels:     b.Labels,
			Title:      b.Title,
			Comments:   b.LenComments,
			Metadata:   b.CreateMetadata,
		}

		if b.AuthorId != "" {
			author, err := env.backend.ResolveIdentityExcerpt(b.AuthorId)
			if err != nil {
				return err
			}
			jsonBug.Author = NewJSONIdentityFromExcerpt(author)
		} else {
			jsonBug.Author = NewJSONIdentityFromLegacyExcerpt(&b.LegacyAuthor)
		}

		if b.AssigneeId != "" {
			assignee, err := env.backend.ResolveIdentityExcerpt(b.AssigneeId)
			if err != nil {
				return err
			}
			jsonBug.Assignee = NewJSONIdentityFromExcerpt(assignee)
		}

		jsonBug.Actors = make([]JSONIdentity, len(b.Actors))
		for i, element := range b.Actors {
			actor, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}
			jsonBug.Actors[i] = NewJSONIdentityFromExcerpt(actor)
		}

		jsonBug.Participants = make([]JSONIdentity, len(b.Participants))
		for i, element := range b.Participants {
			participant, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}
			jsonBug.Participants[i] = NewJSONIdentityFromExcerpt(participant)
		}

		jsonBug.Ccb = make([]JSONCcbInfo, len(b.Ccb))
		for i, element := range b.Ccb {
			user, err := env.backend.ResolveIdentityExcerpt(element.User)
			if err != nil {
				return err
			}
			jsonBug.Ccb[i] = JSONCcbInfo{
				User:   NewJSONIdentityFromExcerpt(user),
				Status: element.Status.String(),
				State:  element.State.String(),
			}

		}

		jsonBugs[i] = jsonBug
	}
	jsonObject, _ := json.MarshalIndent(jsonBugs, "", "    ")
	env.out.Printf("%s\n", jsonObject)
	return nil
}

func lsDefaultFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	termWidth, _, err := text.GetTermDim()
	if err != nil {
		return err
	}

	var titleWidth int
	var fullTerm bool = true

	// Adjust the minimum term width for the minimum width of the terminal that displays the full output
	const minTermWidth = 90

	const statusWidth = 10
	const repoWidth = 12
	const authorWidth = 15
	const assigneeWidth = 15
	const commentCountWidth = 4

	if termWidth >= minTermWidth {
		const paddingWidth = 9 // speech bubble (2) + spaces
		titleWidth = termWidth - (entity.HumanIdLength + statusWidth + repoWidth + authorWidth + assigneeWidth + commentCountWidth + paddingWidth)
	} else {
		fullTerm = false
		const paddingWidth = 2 // spaces
		titleWidth = termWidth - (entity.HumanIdLength + statusWidth + paddingWidth)
	}

	for _, b := range bugExcerpts {
		var authorName string
		if b.AuthorId != "" {
			author, err := env.backend.ResolveIdentityExcerpt(b.AuthorId)
			if err != nil {
				authorName = "<missing author data>"
			} else {
				authorName = author.DisplayName()
			}
		} else {
			authorName = b.LegacyAuthor.DisplayName()
		}

		assigneeName := "UNASSIGNED"
		if b.AssigneeId != "" {
			assignee, err := env.backend.ResolveIdentityExcerpt(b.AssigneeId)
			if err != nil {
				return err
			}
			assigneeName = assignee.DisplayName()
		}

		var labelsTxt strings.Builder
		for _, l := range b.Labels {
			lc256 := l.Color().Term256()
			labelsTxt.WriteString(lc256.Escape())
			labelsTxt.WriteString(" ◼")
			labelsTxt.WriteString(lc256.Unescape())
		}

		// truncate + pad if needed
		labelsFmt := termtext.TruncateMax(labelsTxt.String(), 10)
		titleFmt := termtext.LeftPadMaxLine(strings.TrimSpace(b.Title), titleWidth-termtext.Len(labelsFmt), 0)
		authorFmt := termtext.LeftPadMaxLine(authorName, authorWidth, 0)
		assigneeFmt := termtext.LeftPadMaxLine(assigneeName, assigneeWidth, 0)

		comments := fmt.Sprintf("%*d 💬", commentCountWidth, b.LenComments)
		if b.LenComments >= int(math.Pow(10, commentCountWidth)) {
			comments = strings.Repeat(" ", commentCountWidth-1) + "∞ 💬"
		}

		const kRepoPrefix string = "repo:"
		repo := ""
		for _, l := range b.Labels {
			if strings.HasPrefix(string(l), kRepoPrefix) {
				repo = strings.TrimPrefix(string(l), kRepoPrefix)
			}
		}

		if fullTerm {
			env.out.Printf("%s %s %s %-*s %s %s %s\n",
				colors.Cyan(b.Id.Human()),
				termtext.LeftPadMaxLine(colors.Yellow(b.Status), statusWidth, 0),
				termtext.LeftPadMaxLine(colors.Green(repo), repoWidth, 0),
				titleWidth,
				titleFmt+labelsFmt,
				colors.Magenta(authorFmt),
				colors.Blue(assigneeFmt),
				comments,
			)
		} else {
			env.out.Printf("%s %s %-*s\n",
				colors.Cyan(b.Id.Human()),
				termtext.LeftPadMaxLine(colors.Yellow(b.Status), statusWidth, 0),
				titleWidth,
				titleFmt+labelsFmt,
			)
		}
	}
	return nil
}

func lsPlainFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	for _, b := range bugExcerpts {
		env.out.Printf("%s [%s] %s\n", b.Id.Human(), b.Status, strings.TrimSpace(b.Title))
	}
	return nil
}

func lsOrgmodeFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	// see https://orgmode.org/manual/Tags.html
	orgTagRe := regexp.MustCompile("[^[:alpha:]_@]")
	formatTag := func(l bug.Label) string {
		return orgTagRe.ReplaceAllString(l.String(), "_")
	}

	formatTime := func(time time.Time) string {
		return time.Format("[2006-01-02 Mon 15:05]")
	}

	env.out.Println("#+TODO: OPEN | CLOSED")

	for _, b := range bugExcerpts {
		status := strings.ToUpper(b.Status.String())

		var title string
		if link, ok := b.CreateMetadata["github-url"]; ok {
			title = fmt.Sprintf("[[%s][%s]]", link, b.Title)
		} else {
			title = b.Title
		}

		var name string
		if b.AuthorId != "" {
			author, err := env.backend.ResolveIdentityExcerpt(b.AuthorId)
			if err != nil {
				return err
			}
			name = author.DisplayName()
		} else {
			name = b.LegacyAuthor.DisplayName()
		}

		var labels strings.Builder
		labels.WriteString(":")
		for i, l := range b.Labels {
			if i > 0 {
				labels.WriteString(":")
			}
			labels.WriteString(formatTag(l))
		}
		labels.WriteString(":")

		env.out.Printf("* %-6s %s %s %s: %s %s\n",
			status,
			b.Id.Human(),
			formatTime(b.CreateTime()),
			name,
			title,
			labels.String(),
		)

		env.out.Printf("** Last Edited: %s\n", formatTime(b.EditTime()))

		env.out.Printf("** Actors:\n")
		for _, element := range b.Actors {
			actor, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}

			env.out.Printf(": %s %s\n",
				actor.Id.Human(),
				actor.DisplayName(),
			)
		}

		env.out.Printf("** Participants:\n")
		for _, element := range b.Participants {
			participant, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}

			env.out.Printf(": %s %s\n",
				participant.Id.Human(),
				participant.DisplayName(),
			)
		}
	}

	return nil
}

// Finish the command flags transformation into the query.Query
func completeQuery(opts *lsOptions) error {
	for _, str := range opts.statusQuery {
		if strings.EqualFold(str, "ALL") {
			opts.query.Status = bug.AllStatuses()
			break
		}
		status, err := bug.StatusFromString(str)
		if err != nil {
			return err
		}
		opts.query.Status = append(opts.query.Status, status)
	}

	for _, no := range opts.noQuery {
		switch no {
		case "label":
			opts.query.NoLabel = true
		default:
			return fmt.Errorf("unknown \"no\" filter %s", no)
		}
	}
	if opts.createBeforeQuery != "" {
		parsedTime, err := parseTime(opts.createBeforeQuery)
		if err != nil {
			return err
		}
		opts.query.CreateBefore = parsedTime
	}
	if opts.createAfterQuery != "" {
		parsedTime, err := parseTime(opts.createAfterQuery)
		if err != nil {
			return err
		}
		opts.query.CreateAfter = parsedTime
	}
	if opts.editBeforeQuery != "" {
		parsedTime, err := parseTime(opts.editBeforeQuery)
		if err != nil {
			return err
		}
		opts.query.EditBefore = parsedTime
	}
	if opts.editAfterQuery != "" {
		parsedTime, err := parseTime(opts.editAfterQuery)
		if err != nil {
			return err
		}
		opts.query.EditAfter = parsedTime
	}

	switch opts.sortBy {
	case "id":
		opts.query.OrderBy = query.OrderById
	case "creation":
		opts.query.OrderBy = query.OrderByCreation
	case "edit":
		opts.query.OrderBy = query.OrderByEdit
	default:
		return fmt.Errorf("unknown sort flag %s", opts.sortBy)
	}

	switch opts.sortDirection {
	case "asc":
		opts.query.OrderDirection = query.OrderAscending
	case "desc":
		opts.query.OrderDirection = query.OrderDescending
	default:
		return fmt.Errorf("unknown sort direction %s", opts.sortDirection)
	}

	return nil
}
