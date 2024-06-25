package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daedaleanai/git-ticket/repository"
)

type SingleLabelConfig struct {
	DeprecationMessage string
}

type Label string

type LabelConfig struct {
	FlatMap    map[Label]SingleLabelConfig
	serialized serializedLabelConfig
}

// LoadLabelConfig attempts to read the labels out of the given repository and store it in configuredLabels
func LoadLabelConfig(repo repository.ClockedRepo) (*LabelConfig, error) {
	labelData, err := GetConfig(repo, "labels")
	if err != nil {
		return nil, fmt.Errorf("unable to read label config: %q", err)
	}

	return parseLabelConfig(labelData)
}

func parseLabelConfig(data []byte) (*LabelConfig, error) {
	serializedConfig := serializedLabelConfig{}
	err := json.Unmarshal(data, &serializedConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall labels: %q", err)
	}

	configLabelMap := make(map[Label]SingleLabelConfig)
	for _, labelConfig := range serializedConfig.Labels {
		for _, label := range labelConfig.Labels() {
			if _, ok := configLabelMap[Label(label.Name)]; ok {
				return nil, fmt.Errorf("Duplicated rule for label %s in configuration", label.Name)
			}

			configLabelMap[Label(label.Name)] =
				SingleLabelConfig{
					DeprecationMessage: label.DeprecationMessage,
				}
		}
	}

	// Store the labels
	return &LabelConfig{
		FlatMap:    configLabelMap,
		serialized: serializedConfig,
	}, nil
}

// GetLabelConfig returns the configuration of the given label.
// It will return nil if the label does not exist.
// It will return an error if reading the list of known labels fails
func (l *LabelConfig) GetLabelConfig(label Label) (*SingleLabelConfig, error) {
	if config, ok := l.FlatMap[label]; ok {
		return &config, nil
	}
	return nil, fmt.Errorf("Label %s does not exist", label)
}

// AppendLabelToConfiguration appends a given label to the label store, turning it into a valid label.
// Note that this function does not persistently store it in the configuration.
// Obtain the serialized label configuration and the key in the configuration using LabelStoreData.
func (c *LabelConfig) AppendLabelToConfiguration(label Label) error {
	parts := strings.Split(string(label), ":")

	curPrefixLevel := &c.serialized.Labels
	prefixes := parts[:len(parts)-1]
	for i, curPrefix := range prefixes {
		var targetCompoundPrefix *compoundLabelConfig = nil
		for _, knownLabelConfig := range *curPrefixLevel {
			knownPrefixConfig, ok := knownLabelConfig.(*compoundLabelConfig)
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
			targetCompoundPrefix = &compoundLabelConfig{Prefix: curPrefix}
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

		if compound, ok := curItem.(*compoundLabelConfig); ok && compound.Prefix == lastName {
			conflictingName := strings.Join(parts, ":")
			return fmt.Errorf("A label with name %s is already allocated", conflictingName)
		}
	}

	// At this point we know the label is not allocated and can allocate it
	*curPrefixLevel = append(*curPrefixLevel, &simpleLabelConfig{Name: lastName})

	// Finally add it to the config label map
	c.FlatMap[label] = SingleLabelConfig{}

	return nil
}

// LabelStoreData retrieves the serialized JSON form of the label store, along with the key in the configuration store.
func (c *LabelConfig) Store(repo repository.ClockedRepo) error {
	serialized, err := json.MarshalIndent(c.serialized, "", "  ")
	if err != nil {
		return err
	}

	err = SetConfig(repo, "labels", serialized)
	if err != nil {
		return fmt.Errorf("Unable to store label configuration persistently: %s", err)
	}
	return nil
}

type labelConfigInterface interface {
	// Returns an array of simpleLabelConfig's by recursively expanding all compoundlabelConfig's.
	Labels() []simpleLabelConfig
}

type simpleLabelConfig struct {
	Name               string `json:"name"`
	DeprecationMessage string `json:"deprecationMessage"`
}

type compoundLabelConfig struct {
	Prefix             string                 `json:"prefix"`
	Inner              []labelConfigInterface `json:"labels"`
	DeprecationMessage string                 `json:"deprecationMessage"`
}

// This type is internal and used for the internal store of the labels.
// for convenience it is converted to a LabelConfigMap for consumption within
// the rest of the git-ticket code.
type serializedLabelConfig struct {
	Labels []labelConfigInterface `json:"labels"`
}

func (l *compoundLabelConfig) Labels() []simpleLabelConfig {
	labels := []simpleLabelConfig{}
	for _, labelConfig := range l.Inner {
		innerLabels := labelConfig.Labels()
		for _, innerLabel := range innerLabels {
			if l.DeprecationMessage != "" {
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

// unmarshallLabelConfigInterface unmarshalls the given data as a labelConfigInterface.
func unmarshallLabelConfigInterface(data []byte) (labelConfigInterface, error) {
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
		var compound compoundLabelConfig
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

func (simple simpleLabelConfig) MarshalJSON() ([]byte, error) {
	if simple.DeprecationMessage == "" {
		return json.Marshal(simple.Name)
	}

	raw := struct {
		Name               string `json:"name"`
		DeprecationMessage string `json:"deprecationMessage"`
	}{
		Name:               simple.Name,
		DeprecationMessage: simple.DeprecationMessage,
	}

	return json.Marshal(raw)
}

func (c *serializedLabelConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		Labels []json.RawMessage
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	c.Labels = []labelConfigInterface{}
	for _, message := range raw.Labels {
		config, err := unmarshallLabelConfigInterface(message)
		if err != nil {
			return err
		}
		c.Labels = append(c.Labels, config)
	}

	return nil
}

func (c *compoundLabelConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		Prefix             string
		Labels             []json.RawMessage
		DeprecationMessage string
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	c.Prefix = raw.Prefix
	c.DeprecationMessage = raw.DeprecationMessage
	c.Inner = []labelConfigInterface{}
	for _, message := range raw.Labels {
		config, err := unmarshallLabelConfigInterface(message)
		if err != nil {
			return err
		}
		c.Inner = append(c.Inner, config)
	}

	return nil
}
