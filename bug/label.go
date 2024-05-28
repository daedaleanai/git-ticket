package bug

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"strings"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/text"

	fcolor "github.com/fatih/color"
)

type Label string

func (l Label) String() string {
	return string(l)
}

// RGBA from a Label computed in a deterministic way
func (l Label) Color() LabelColor {
	// colors from: https://material-ui.com/style/color/
	colors := []LabelColor{
		{R: 244, G: 67, B: 54, A: 255},   // red
		{R: 233, G: 30, B: 99, A: 255},   // pink
		{R: 156, G: 39, B: 176, A: 255},  // purple
		{R: 103, G: 58, B: 183, A: 255},  // deepPurple
		{R: 63, G: 81, B: 181, A: 255},   // indigo
		{R: 33, G: 150, B: 243, A: 255},  // blue
		{R: 3, G: 169, B: 244, A: 255},   // lightBlue
		{R: 0, G: 188, B: 212, A: 255},   // cyan
		{R: 0, G: 150, B: 136, A: 255},   // teal
		{R: 76, G: 175, B: 80, A: 255},   // green
		{R: 139, G: 195, B: 74, A: 255},  // lightGreen
		{R: 205, G: 220, B: 57, A: 255},  // lime
		{R: 255, G: 235, B: 59, A: 255},  // yellow
		{R: 255, G: 193, B: 7, A: 255},   // amber
		{R: 255, G: 152, B: 0, A: 255},   // orange
		{R: 255, G: 87, B: 34, A: 255},   // deepOrange
		{R: 121, G: 85, B: 72, A: 255},   // brown
		{R: 158, G: 158, B: 158, A: 255}, // grey
		{R: 96, G: 125, B: 139, A: 255},  // blueGrey
	}

	id := 0
	hash := sha256.Sum256([]byte(l))
	for _, char := range hash {
		id = (id + int(char)) % len(colors)
	}

	return colors[id]
}

func (l Label) Validate() error {
	str := string(l)

	if text.Empty(str) {
		return fmt.Errorf("empty")
	}

	if strings.Contains(str, "\n") {
		return fmt.Errorf("should be a single line")
	}

	if !text.Safe(str) {
		return fmt.Errorf("not fully printable")
	}

	return nil
}

type LabelColor color.RGBA

func (lc LabelColor) RGBA() color.RGBA {
	return color.RGBA(lc)
}

func (lc LabelColor) Term256() Term256 {
	red := Term256(lc.R) * 6 / 256
	green := Term256(lc.G) * 6 / 256
	blue := Term256(lc.B) * 6 / 256

	return red*36 + green*6 + blue + 16
}

type Term256 int

func (t Term256) Escape() string {
	if fcolor.NoColor {
		return ""
	}
	return fmt.Sprintf("\x1b[38;5;%dm", t)
}

func (t Term256) Unescape() string {
	if fcolor.NoColor {
		return ""
	}
	return "\x1b[0m"
}

func (l Label) IsChecklist() bool {
	return strings.HasPrefix(string(l), "checklist:")
}

func (l Label) IsWorkflow() bool {
	return strings.HasPrefix(string(l), "workflow:")
}

type simpleLabelConfig struct {
	Name               string `json:"name"`
	Deprecated         bool   `json:"deprecated"`
	DeprecationMessage string `json:"deprecationMessage"`
}

type compoundlabelConfig struct {
	Prefix             string                 `json:"prefix"`
	Inner              []labelConfigInterface `json:"labels"`
	Deprecated         bool                   `json:"deprecated"`
	DeprecationMessage string                 `json:"deprecationMessage"`
}

type labelConfigInterface interface {
	Labels() []simpleLabelConfig
}

type serializedLabelConfig struct {
	Labels []labelConfigInterface `json:"labels"`
}

type LabelConfig struct {
	Deprecated         bool
	DeprecationMessage string
}

type LabelConfigMap map[Label]LabelConfig

func (l *compoundlabelConfig) Labels() []simpleLabelConfig {
	labels := []simpleLabelConfig{}
	for _, labelConfig := range l.Inner {
		innerLabels := labelConfig.Labels()
		for _, innerLabel := range innerLabels {
			if l.Deprecated {
				innerLabel.Deprecated = true
				innerLabel.DeprecationMessage = l.DeprecationMessage
			}
			innerLabel.Name = l.Prefix + ":" + innerLabel.Name
			labels = append(labels, innerLabel)
		}
	}
	return labels
}

func (l *simpleLabelConfig) Labels() []simpleLabelConfig {
	return []simpleLabelConfig{*l}
}

func UnmarshallLabelConfigInterface(data []byte) (labelConfigInterface, error) {
	// Try to unmarshall as a regular string
	var s string
	err := json.Unmarshal(data, &s)
	if err == nil {
		return &simpleLabelConfig{Name: s}, nil
	}

	// If it is a struct we need to figure out if it is simple or compound
	var raw map[string]json.RawMessage
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshall labelConfigInterface. The byte array is not a string or object: %s", err)
	}

	if _, ok := raw["prefix"]; ok {
		var compound compoundlabelConfig
		err = json.Unmarshal(data, &compound)
		if err != nil {
			return nil, fmt.Errorf("Unable to unmarshall compound label config: %s", err)
		}
		return &compound, nil
	}

	if _, ok := raw["name"]; ok {
		var simple simpleLabelConfig
		err = json.Unmarshal(data, &simple)
		if err != nil {
			return nil, fmt.Errorf("Unable to unmarshall simple label config: %s", err)
		}
		return &simple, nil
	}

	return nil, fmt.Errorf("Unable to unmarshall label config: %s", string(data))
}

func (op *serializedLabelConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		Labels []json.RawMessage
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	op.Labels = []labelConfigInterface{}
	for _, message := range raw.Labels {
		config, err := UnmarshallLabelConfigInterface(message)
		if err != nil {
			return err
		}
		op.Labels = append(op.Labels, config)
	}

	return nil
}

func (op *compoundlabelConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		Prefix             string
		Labels             []json.RawMessage
		Deprecated         bool
		DeprecationMessage string
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	op.Prefix = raw.Prefix
	op.Deprecated = raw.Deprecated
	op.DeprecationMessage = raw.DeprecationMessage
	op.Inner = []labelConfigInterface{}
	for _, message := range raw.Labels {
		config, err := UnmarshallLabelConfigInterface(message)
		if err != nil {
			return err
		}
		op.Inner = append(op.Inner, config)
	}

	return nil
}

var configuredLabels LabelConfigMap = nil
var labelStore *serializedLabelConfig = nil

func parseConfiguredLabels(data []byte) (*LabelConfigMap, *serializedLabelConfig, error) {
	serializedConfig := serializedLabelConfig{}
	err := json.Unmarshal(data, &serializedConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load ccb: %q", err)
	}

	configLabelMap := make(LabelConfigMap)
	for _, labelConfig := range serializedConfig.Labels {
		for _, label := range labelConfig.Labels() {
			if _, ok := configLabelMap[Label(label.Name)]; ok {
				return nil, nil, fmt.Errorf("Duplicated rule for label %s in configuration", label.Name)
			}

			configLabelMap[Label(label.Name)] =
				LabelConfig{
					Deprecated:         label.Deprecated,
					DeprecationMessage: label.DeprecationMessage,
				}
		}
	}

	return &configLabelMap, &serializedConfig, nil
}

// readConfiguredLabels attempts to read the labels out of the current repository and store it in configuredLabels
func readConfiguredLabels() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get the current working directory: %q", err)
	}

	repo, err := repository.NewGitRepo(cwd, []repository.ClockLoader{ClockLoader})
	if err == repository.ErrNotARepo {
		return fmt.Errorf("must be run from within a git repo")
	}

	labelData, err := config.GetConfig(repo, "labels")
	if err != nil {
		return fmt.Errorf("unable to read label config: %q", err)
	}

	configLabelMap, serializedConfig, err := parseConfiguredLabels(labelData)
	if err != nil {
		return err
	}

	// Store the labels
	configuredLabels = *configLabelMap
	labelStore = serializedConfig
	return nil
}

// IsKnownLabel returns true if the given label belongs to the list of known (configured) labels.
// It may return an error if reading the list of known labels fails
func GetLabelConfig(label Label) (*LabelConfig, error) {
	if configuredLabels == nil {
		if err := readConfiguredLabels(); err != nil {
			return nil, err
		}
	}
	if config, ok := configuredLabels[label]; !ok {
		return &config, nil
	}
	return nil, nil
}

// ListLabels returns the map of configured labels
func ListLabels() (LabelConfigMap, error) {
	if configuredLabels == nil {
		if err := readConfiguredLabels(); err != nil {
			return nil, err
		}
	}
	return configuredLabels, nil
}

func AppendLabelToConfiguration(label Label) error {
	parts := strings.Split(string(label), ":")

	if label.IsWorkflow() {
		return fmt.Errorf("Workflow labels are not part of the configuration. Modify git-ticket source code at bug/workflow.go instead.")
	}

	if label.IsChecklist() {
		return fmt.Errorf("Checklist labels are not part of the configuration. Use `git ticket config set checklists` instead.")
	}

	if configuredLabels == nil {
		if err := readConfiguredLabels(); err != nil {
			return err
		}
	}

	curPrefixLevel := &labelStore.Labels
	prefixes := parts[:len(parts)-1]
	for i, curPrefix := range prefixes {
		var targetCompoundPrefix *compoundlabelConfig = nil
		for _, knownLabelConfig := range *curPrefixLevel {
			knownPrefixConfig, ok := knownLabelConfig.(*compoundlabelConfig)
			if !ok {
				simpleLabel := knownLabelConfig.(*simpleLabelConfig)
				if simpleLabel.Name == curPrefix {
					conflictingName := strings.Join(prefixes[:i+1], ":")
					return fmt.Errorf("A label with name %s is already allocated", conflictingName)
				}
				continue
			}

			if knownPrefixConfig.Prefix == curPrefix {
				targetCompoundPrefix = knownPrefixConfig
				break
			}
		}

		if targetCompoundPrefix == nil {
			// Create the label config for the given prefix
			targetCompoundPrefix = &compoundlabelConfig{Prefix: curPrefix}
			*curPrefixLevel = append(*curPrefixLevel, targetCompoundPrefix)
		}

		curPrefixLevel = &targetCompoundPrefix.Inner
	}

	lastName := parts[len(parts)-1]
	for _, curItem := range *curPrefixLevel {
		if simple, ok := curItem.(*simpleLabelConfig); ok && simple.Name == lastName {
			conflictingName := strings.Join(parts, ":")
			return fmt.Errorf("A label with name %s is already allocated", conflictingName)
		}

		if compound, ok := curItem.(*compoundlabelConfig); ok && compound.Prefix == lastName {
			conflictingName := strings.Join(parts, ":")
			return fmt.Errorf("A label with name %s is already allocated", conflictingName)
		}
	}

	// At this point we know the label is not allocated and can allocate it
	*curPrefixLevel = append(*curPrefixLevel, &simpleLabelConfig{Name: lastName})

	// Finally add it to the config label map
	configuredLabels[label] = LabelConfig{}

	return nil
}

func LabelStoreData() (string, []byte, error) {
	serialized, err := json.Marshal(*labelStore)
	if err != nil {
		return "", nil, err
	}
	return "labels", serialized, nil
}
