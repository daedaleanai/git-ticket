// Inspired by the git-appraise project

// Package input contains helpers to use a text editor as an input for
// various field of a bug
package input

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/pkg/errors"
)

const messageFilename = "BUG_MESSAGE_EDITMSG"
const keyFilename = "KEY_EDITMSG"
const checklistFilename = "CHECKLIST_EDITMSG"
const configFilename = "CONFIG_EDITMSG"

// ErrEmptyMessage is returned when the required message has not been entered
var ErrEmptyMessage = errors.New("empty message")

// ErrEmptyMessage is returned when the required title has not been entered
var ErrEmptyTitle = errors.New("empty title")

const bugTitleCommentTemplate = `%s%s

# Please enter the title and comment message. The first non-empty line will be
# used as the title. Lines starting with '#' will be ignored.
# An empty title aborts the operation.
`

// BugCreateEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract title
// and message.
func BugCreateEditorInput(repo repository.RepoCommon, preTitle string, preMessage string) (string, string, error) {
	if preMessage != "" {
		preMessage = "\n\n" + preMessage
	}

	template := fmt.Sprintf(bugTitleCommentTemplate, preTitle, preMessage)

	raw, err := launchEditorWithTemplate(repo, messageFilename, template)

	if err != nil {
		return "", "", err
	}

	return processCreate(raw)
}

// BugCreateFileInput read from either from a file or from the standard input
// and extract a title and a message
func BugCreateFileInput(fileName string) (string, string, error) {
	raw, err := TextFileInput(fileName)
	if err != nil {
		return "", "", err
	}

	return processCreate(raw)
}

func processCreate(raw string) (string, string, error) {
	lines := strings.Split(raw, "\n")

	var title string
	var buffer bytes.Buffer
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}

		if title == "" {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				title = trimmed
			}
			continue
		}

		buffer.WriteString(line)
		buffer.WriteString("\n")
	}

	if title == "" {
		return "", "", ErrEmptyTitle
	}

	message := strings.TrimSpace(buffer.String())

	return title, message, nil
}

const bugCommentTemplate = `%s

# Please enter the comment message. Lines starting with '#' will be ignored,
# and an empty message aborts the operation.
`

// BugCommentEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a comment.
func BugCommentEditorInput(repo repository.RepoCommon, preMessage string) (string, error) {
	template := fmt.Sprintf(bugCommentTemplate, preMessage)
	raw, err := launchEditorWithTemplate(repo, messageFilename, template)

	if err != nil {
		return "", err
	}

	return processComment(raw)
}

const identityVersionKeyTemplate = `%s

# Please enter the armored key block. Lines starting with '#' will be ignored,
# and an empty message aborts the operation.
`

// IdentityVersionKeyEditorInput will open the default editor in the terminal
// with a template for the user to fill. The file is then processed to extract
// the key.
func IdentityVersionKeyEditorInput(repo repository.RepoCommon, preMessage string) (string, error) {
	template := fmt.Sprintf(identityVersionKeyTemplate, preMessage)
	raw, err := launchEditorWithTemplate(repo, keyFilename, template)

	if err != nil {
		return "", err
	}

	return removeCommentedLines(raw), nil
}

const configTemplate = `
# Please enter your configuration data. Lines starting with '#' will be ignored,
# and an empty message aborts the operation.

%s
`

// ConfigEditorInput will open the default editor in the terminal
// with a template for the user to fill.
func ConfigEditorInput(repo repository.RepoCommon, existingConfig string) (string, error) {
	template := fmt.Sprintf(configTemplate, existingConfig)
	raw, err := launchEditorWithTemplate(repo, configFilename, template)

	if err != nil {
		return "", err
	}

	return processComment(raw)
}

// ConfigFileInput read either from a file or from the standard input
// and extract the config data
func ConfigFileInput(fileName string) (string, error) {
	raw, err := TextFileInput(fileName)
	if err != nil {
		return "", err
	}

	return removeCommentedLines(raw), nil
}

// BugCommentFileInput read either from a file or from the standard input
// and extract a message
func BugCommentFileInput(fileName string) (string, error) {
	raw, err := TextFileInput(fileName)
	if err != nil {
		return "", err
	}

	return processComment(raw)
}

func processComment(raw string) (string, error) {
	message := removeCommentedLines(raw)

	if message == "" {
		return "", ErrEmptyMessage
	}

	return message, nil
}

// removeCommentedLines removes the lines starting with '#' and and
// trims the result.
func removeCommentedLines(raw string) string {
	lines := strings.Split(raw, "\n")

	var buffer bytes.Buffer
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		buffer.WriteString(line)
		buffer.WriteString("\n")
	}

	return strings.TrimSpace(buffer.String())
}

const bugTitleTemplate = `%s

# Please enter the new title. Only one line will used.
# Lines starting with '#' will be ignored, and an empty title aborts the operation.
`

// BugTitleEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a title.
func BugTitleEditorInput(repo repository.RepoCommon, preTitle string) (string, error) {
	template := fmt.Sprintf(bugTitleTemplate, preTitle)
	raw, err := launchEditorWithTemplate(repo, messageFilename, template)

	if err != nil {
		return "", err
	}

	lines := strings.Split(raw, "\n")

	var title string
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		title = trimmed
		break
	}

	if title == "" {
		return "", ErrEmptyTitle
	}

	return title, nil
}

const checklistPreamble = `# %s

# Leave lines starting with '#' unchanged.
# States in [] can be: TBD, PASSED, FAILED or NA. Abbreviations accepted.
# Any text entered between the question and state will be saved as a comment.
# Saving an empty (or invalid format) aborts the operation.

`

// ChecklistEditorInput will open the default editor in the terminal with a
// checklist for the user to fill. The file is then processed to extract the
// comment and status for each question, results are added to checklist.
// Returns bool indicating if anything changed and any error value.
func ChecklistEditorInput(repo repository.RepoCommon, checklist bug.Checklist, ignoreBackup bool) (bool, error) {

	checklistBackupFile := ".git-ticket." + checklist.Label.String() + ".backup"

	var template string

	// If a backup file already exists then load that instead
	backupData, err := ioutil.ReadFile(checklistBackupFile)
	if !ignoreBackup && err == nil {
		template = string(backupData)
	} else {
		template = fmt.Sprintf(checklistPreamble, checklist.Title)
		for sn, s := range checklist.Sections {
			template = template + fmt.Sprintf("#\n#### %s ####\n#\n", s.Title)
			for qn, q := range s.Questions {
				template = template + fmt.Sprintf("# %d.%d : %s\n", sn+1, qn+1, q.Question)
				if len(strings.TrimSpace(q.Comment)) != 0 {
					template = template + fmt.Sprintf("%s\n", q.Comment)
				}
				template = template + fmt.Sprintf("[%s]\n\n", q.State.ShortString())
			}
		}
	}
	raw, err := launchEditorWithTemplate(repo, checklistFilename, template)

	if err != nil {
		return false, err
	}

	err = ioutil.WriteFile(checklistBackupFile, []byte(raw), 0644)
	if err != nil {
		return false, err
	}

	lines := strings.Split(raw, "\n")

	var commentText string
	var inComment bool
	var checklistChanged bool
	nextS := 1
	nextQ := 1

	questionSearch, _ := regexp.Compile(`^# (\d+)\.(\d+) : (\w+)`)
	stateSearch, _ := regexp.Compile(`^\[(.+)\]$`)

	for l, line := range lines {
		if !inComment {
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}

			if questionSearch.MatchString(line) {
				// check question number and reset comment
				matches := questionSearch.FindStringSubmatch(line)
				if thisS, err := strconv.Atoi(matches[1]); err != nil || thisS != nextS {
					// unexpected section number
					return checklistChanged, fmt.Errorf("checklist parse error (section number), line %d", l)
				}
				if thisQ, err := strconv.Atoi(matches[2]); err != nil || thisQ != nextQ {
					// unexpected question number
					return checklistChanged, fmt.Errorf("checklist parse error (question number), line %d", l)
				}
				inComment = true
				commentText = ""
			} else if nextQ != 1 {
				// next question line missing
				return checklistChanged, fmt.Errorf("checklist parse error (question line), line %d", l)
			}
		} else {
			if stateSearch.MatchString(line) {
				newState, err := bug.StateFromString(stateSearch.FindStringSubmatch(line)[1])
				if err != nil {
					// something is wrong with the format
					return checklistChanged, fmt.Errorf("checklist parse error (invalid state), line %d", l)
				}
				// check and save comment
				strippedCommentText := strings.TrimSuffix(commentText, "\n")
				if checklist.Sections[nextS-1].Questions[nextQ-1].Comment != strippedCommentText {
					checklist.Sections[nextS-1].Questions[nextQ-1].Comment = strippedCommentText
					checklistChanged = true
				}
				// check and save state
				if checklist.Sections[nextS-1].Questions[nextQ-1].State != newState {
					checklist.Sections[nextS-1].Questions[nextQ-1].State = newState
					checklistChanged = true
				}
				nextQ++
				if nextQ > len(checklist.Sections[nextS-1].Questions) {
					nextS++
					nextQ = 1
				}
				inComment = false
			} else {
				// we're still in the comment section
				commentText = commentText + line + "\n"
			}
		}
	}

	if nextS != len(checklist.Sections)+1 {
		return checklistChanged, fmt.Errorf("checklist parse error, section/question count")
	}

	os.Remove(checklistBackupFile)

	return checklistChanged, nil
}

const queryTemplate = `%s

# Please edit the bug query.
# Lines starting with '#' will be ignored, and an empty query aborts the operation.
#
# Example: status:open author:"rené descartes" sort:edit
#
# Valid filters are:
#
# - status:open, status:closed
# - author:<query>
# - title:<title>
# - label:<label>
# - no:label
#
# Sorting
#
# - sort:id, sort:id-desc, sort:id-asc
# - sort:creation, sort:creation-desc, sort:creation-asc
# - sort:edit, sort:edit-desc, sort:edit-asc
#
# Notes
#
# - queries are case insensitive.
# - you can combine as many qualifiers as you want.
# - you can use double quotes for multi-word search terms (ex: author:"René Descartes")
`

// QueryEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a query.
func QueryEditorInput(repo repository.RepoCommon, preQuery string) (string, error) {
	template := fmt.Sprintf(queryTemplate, preQuery)
	raw, err := launchEditorWithTemplate(repo, messageFilename, template)

	if err != nil {
		return "", err
	}

	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		return trimmed, nil
	}

	return "", nil
}

// launchEditorWithTemplate will launch an editor as launchEditor do, but with a
// provided template.
func launchEditorWithTemplate(repo repository.RepoCommon, fileName string, template string) (string, error) {
	path := fmt.Sprintf("%s/%s", repo.GetPath(), fileName)

	err := ioutil.WriteFile(path, []byte(template), 0644)

	if err != nil {
		return "", err
	}

	return launchEditor(repo, fileName)
}

// launchEditor launches the default editor configured for the given repo. This
// method blocks until the editor command has returned.
//
// The specified filename should be a temporary file and provided as a relative path
// from the repo (e.g. "FILENAME" will be converted to "[<reporoot>/].git/FILENAME"). This file
// will be deleted after the editor is closed and its contents have been read.
//
// This method returns the text that was read from the temporary file, or
// an error if any step in the process failed.
func launchEditor(repo repository.RepoCommon, fileName string) (string, error) {
	path := fmt.Sprintf("%s/%s", repo.GetPath(), fileName)
	defer os.Remove(path)

	editor, err := repo.GetCoreEditor()
	if err != nil {
		return "", fmt.Errorf("Unable to detect default git editor: %v\n", err)
	}

	cmd, err := startInlineCommand(editor, path)
	if err != nil {
		// Running the editor directly did not work. This might mean that
		// the editor string is not a path to an executable, but rather
		// a shell command (e.g. "emacsclient --tty"). As such, we'll try
		// to run the command through bash, and if that fails, try with sh
		args := []string{"-c", fmt.Sprintf("%s %q", editor, path)}
		cmd, err = startInlineCommand("bash", args...)
		if err != nil {
			cmd, err = startInlineCommand("sh", args...)
		}
	}
	if err != nil {
		return "", fmt.Errorf("Unable to start editor: %v\n", err)
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("Editing finished with error: %v\n", err)
	}

	output, err := ioutil.ReadFile(path)

	if err != nil {
		return "", fmt.Errorf("Error reading edited file: %v\n", err)
	}

	return string(output), err
}

// TextFileInput loads and returns the contents of a given file. If - is passed
// through, much like git, it will read from stdin. This can be piped data,
// unless there is a tty in which case the user will be prompted to enter a
// message.
func TextFileInput(fileName string) (string, error) {
	if fileName == "-" {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return "", fmt.Errorf("Error reading from stdin: %v\n", err)
		}
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// There is no tty. This will allow us to read piped data instead.
			output, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return "", fmt.Errorf("Error reading from stdin: %v\n", err)
			}
			return string(output), err
		}

		fmt.Printf("(reading comment from standard input)\n")
		var output bytes.Buffer
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			output.Write(s.Bytes())
			output.WriteRune('\n')
		}
		return output.String(), nil
	}

	output, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("Error reading file: %v\n", err)
	}
	return string(output), err
}

func startInlineCommand(command string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	return cmd, err
}
