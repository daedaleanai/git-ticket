package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLabelConfigUnmarshall(t *testing.T) {
	labelConfigJson := `
{
  "labels": [
    "simple-label",
    {
        "name": "simple-label-but-deprecated",
        "deprecationMessage": "I'm sure there is a reason"
    },
    {
      "prefix": "impact",
      "labels": [
        "vyper-sdd",
        {
            "name": "dep",
            "deprecationMessage": "There's no reason"
        },
        {
            "prefix": "another",
            "labels": [],
            "deprecationMessage": ""
        }
      ]
    }
  ]
}
`

	config, err := parseLabelConfig([]byte(labelConfigJson))
	if err != nil {
		t.Fatal("Unable to unmarshall label configuration: ", err)
	}

	assert.Len(t, config.serialized.Labels, 3)
	assert.IsType(t, &simpleLabelConfig{}, config.serialized.Labels[0])
	item0 := config.serialized.Labels[0].(*simpleLabelConfig)
	assert.Equal(t, item0.Name, "simple-label")
	assert.Equal(t, item0.DeprecationMessage, "")

	assert.IsType(t, &simpleLabelConfig{}, config.serialized.Labels[1])
	item1 := config.serialized.Labels[1].(*simpleLabelConfig)
	assert.Equal(t, item1.Name, "simple-label-but-deprecated")
	assert.Equal(t, item1.DeprecationMessage, "I'm sure there is a reason")

	assert.IsType(t, &compoundLabelConfig{}, config.serialized.Labels[2])
	item2 := config.serialized.Labels[2].(*compoundLabelConfig)
	assert.Equal(t, item2.Prefix, "impact")
	assert.Equal(t, item2.DeprecationMessage, "")

	assert.Len(t, item2.Inner, 3)

	assert.IsType(t, &simpleLabelConfig{}, item2.Inner[0])
	innerItem0 := item2.Inner[0].(*simpleLabelConfig)
	assert.Equal(t, innerItem0.Name, "vyper-sdd")
	assert.Equal(t, innerItem0.DeprecationMessage, "")

	assert.IsType(t, &simpleLabelConfig{}, item2.Inner[1])
	innerItem1 := item2.Inner[1].(*simpleLabelConfig)
	assert.Equal(t, innerItem1.Name, "dep")
	assert.Equal(t, innerItem1.DeprecationMessage, "There's no reason")

	assert.IsType(t, &compoundLabelConfig{}, item2.Inner[2])
	innerItem2 := item2.Inner[2].(*compoundLabelConfig)
	assert.Equal(t, innerItem2.Prefix, "another")
	assert.Len(t, innerItem2.Inner, 0)
	assert.Equal(t, innerItem2.DeprecationMessage, "")
}

func TestLabelMappingConfigUnmarshall(t *testing.T) {
	labelConfigJson := `
{
    "labelMapping": {
		"impact:psac" : { "pCCB": ["swTeam"], "sCCB": ["certTeam"], "checklists": ["checklist:swplan"] },
		"impact:sdd" : { "pCCB": ["swTeam"], "sCCB": ["swTeam", "hwTeam"], "checklists": ["checklist:swdesign"] }
    }
}
`

	config, err := parseLabelConfig([]byte(labelConfigJson))
	if err != nil {
		t.Fatal("Unable to unmarshall label configuration: ", err)
	}

	impact := config.serialized.LabelMapping

	assert.Len(t, impact, 2)
	assert.Contains(t, impact, Label("impact:psac"))
	assert.Contains(t, impact, Label("impact:sdd"))

	assert.Equal(t, CcbAndChecklistConfig{
		PrimaryCcbTeams:    []string{"swTeam"},
		SecondaryCcbTeams:  []string{"certTeam"},
		RequiredChecklists: []string{"checklist:swplan"},
	}, impact["impact:psac"])

	assert.Equal(t, CcbAndChecklistConfig{
		PrimaryCcbTeams:    []string{"swTeam"},
		SecondaryCcbTeams:  []string{"swTeam", "hwTeam"},
		RequiredChecklists: []string{"checklist:swdesign"},
	}, impact["impact:sdd"])
}

func TestLabelConfigPlainMap(t *testing.T) {
	labelConfigJson := `
{
  "labels": [
    "simple-label",
    {
        "name": "simple-label-but-deprecated",
        "deprecationMessage": "I'm sure there is a reason"
    },
    {
      "prefix": "impact",
      "labels": [
        "vyper-sdd",
        {
            "name": "dep",
            "deprecationMessage": "There's no reason"
        },
        {
            "prefix": "another",
            "labels": ["one", "two"],
            "deprecationMessage": ""
        }
      ]
    }
  ]
}
`

	config, err := parseLabelConfig([]byte(labelConfigJson))
	if err != nil {
		t.Fatal("Unable to unmarshall label configuration: ", err)
	}

	assert.Len(t, config.FlatMap, 6)
	assert.Contains(t, config.FlatMap, Label("simple-label"))
	assert.Contains(t, config.FlatMap, Label("simple-label-but-deprecated"))
	assert.Contains(t, config.FlatMap, Label("impact:vyper-sdd"))
	assert.Contains(t, config.FlatMap, Label("impact:dep"))
	assert.Contains(t, config.FlatMap, Label("impact:another:one"))
	assert.Contains(t, config.FlatMap, Label("impact:another:two"))
}

func TestLabelConfigSerialize(t *testing.T) {
	labelStore := serializedLabelConfig{
		Labels: []labelConfigInterface{
			&simpleLabelConfig{Name: "simple-label"},
			&simpleLabelConfig{Name: "simple-label-but-deprecated", DeprecationMessage: "I'm sure there is a reason"},
			&compoundLabelConfig{
				Prefix: "impact",
				Inner: []labelConfigInterface{
					&simpleLabelConfig{Name: "vyper-sdd"},
					&simpleLabelConfig{Name: "dep", DeprecationMessage: "There's no reason"},
					&compoundLabelConfig{
						Prefix: "another",
						Inner: []labelConfigInterface{
							&simpleLabelConfig{Name: "one"},
							&simpleLabelConfig{Name: "two"},
						},
					},
				},
			},
		},
	}

	serialized, err := json.Marshal(labelStore)
	if err != nil {
		t.Fatal("Unable to marshall label configuration: ", err)
	}

	var deserialized serializedLabelConfig
	err = json.Unmarshal(serialized, &deserialized)
	if err != nil {
		t.Fatal("Unable to unmarshall label configuration: ", err)
	}

	assert.Equal(t, labelStore, deserialized)
}
