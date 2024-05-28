package bug

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelRGBA(t *testing.T) {
	rgba := Label("test1").Color()
	expected := LabelColor{R: 0, G: 150, B: 136, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBASimilar(t *testing.T) {
	rgba := Label("test2").Color()
	expected := LabelColor{R: 3, G: 169, B: 244, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAReverse(t *testing.T) {
	rgba := Label("tset").Color()
	expected := LabelColor{R: 63, G: 81, B: 181, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAEqual(t *testing.T) {
	color1 := Label("test").Color()
	color2 := Label("test").Color()

	require.Equal(t, color1, color2)
}

func TestLabelConfigUnmarshall(t *testing.T) {
	labelConfigJson := `
{
  "labels": [
    "simple-label",
    {
        "name": "simple-label-but-deprecated",
        "deprecated": true,
        "deprecationMessage": "I'm sure there is a reason"
    },
    {
      "prefix": "impact",
      "labels": [
        "vyper-sdd",
        {
            "name": "dep",
            "deprecated": true,
            "deprecationMessage": "There's no reason"
        },
        {
            "prefix": "another",
            "labels": [],
            "deprecated": false,
            "deprecationMessage": ""
        }
      ]
    }
  ]
}
`

	_, serializedConfig, err := parseConfiguredLabels([]byte(labelConfigJson))
	if err != nil {
		t.Fatal("Unable to unmarshall label configuration: ", err)
	}

	assert.Len(t, serializedConfig.Labels, 3)
	assert.IsType(t, &simpleLabelConfig{}, serializedConfig.Labels[0])
	item0 := serializedConfig.Labels[0].(*simpleLabelConfig)
	assert.Equal(t, item0.Name, "simple-label")
	assert.Equal(t, item0.Deprecated, false)
	assert.Equal(t, item0.DeprecationMessage, "")

	assert.IsType(t, &simpleLabelConfig{}, serializedConfig.Labels[1])
	item1 := serializedConfig.Labels[1].(*simpleLabelConfig)
	assert.Equal(t, item1.Name, "simple-label-but-deprecated")
	assert.Equal(t, item1.Deprecated, true)
	assert.Equal(t, item1.DeprecationMessage, "I'm sure there is a reason")

	assert.IsType(t, &compoundLabelConfig{}, serializedConfig.Labels[2])
	item2 := serializedConfig.Labels[2].(*compoundLabelConfig)
	assert.Equal(t, item2.Prefix, "impact")
	assert.Equal(t, item2.Deprecated, false)
	assert.Equal(t, item2.DeprecationMessage, "")

	assert.Len(t, item2.Inner, 3)

	assert.IsType(t, &simpleLabelConfig{}, item2.Inner[0])
	innerItem0 := item2.Inner[0].(*simpleLabelConfig)
	assert.Equal(t, innerItem0.Name, "vyper-sdd")
	assert.Equal(t, innerItem0.Deprecated, false)
	assert.Equal(t, innerItem0.DeprecationMessage, "")

	assert.IsType(t, &simpleLabelConfig{}, item2.Inner[1])
	innerItem1 := item2.Inner[1].(*simpleLabelConfig)
	assert.Equal(t, innerItem1.Name, "dep")
	assert.Equal(t, innerItem1.Deprecated, true)
	assert.Equal(t, innerItem1.DeprecationMessage, "There's no reason")

	assert.IsType(t, &compoundLabelConfig{}, item2.Inner[2])
	innerItem2 := item2.Inner[2].(*compoundLabelConfig)
	assert.Equal(t, innerItem2.Prefix, "another")
	assert.Len(t, innerItem2.Inner, 0)
	assert.Equal(t, innerItem2.Deprecated, false)
	assert.Equal(t, innerItem2.DeprecationMessage, "")
}

func TestLabelConfigPlainMap(t *testing.T) {
	labelConfigJson := `
{
  "labels": [
    "simple-label",
    {
        "name": "simple-label-but-deprecated",
        "deprecated": true,
        "deprecationMessage": "I'm sure there is a reason"
    },
    {
      "prefix": "impact",
      "labels": [
        "vyper-sdd",
        {
            "name": "dep",
            "deprecated": true,
            "deprecationMessage": "There's no reason"
        },
        {
            "prefix": "another",
            "labels": ["one", "two"],
            "deprecated": false,
            "deprecationMessage": ""
        }
      ]
    }
  ]
}
`

	configMap, _, err := parseConfiguredLabels([]byte(labelConfigJson))
	if err != nil {
		t.Fatal("Unable to unmarshall label configuration: ", err)
	}

	assert.Len(t, *configMap, 6)
	assert.Contains(t, *configMap, Label("simple-label"))
	assert.Contains(t, *configMap, Label("simple-label-but-deprecated"))
	assert.Contains(t, *configMap, Label("impact:vyper-sdd"))
	assert.Contains(t, *configMap, Label("impact:dep"))
	assert.Contains(t, *configMap, Label("impact:another:one"))
	assert.Contains(t, *configMap, Label("impact:another:two"))
}

func TestLabelConfigSerialize(t *testing.T) {
	labelStore := serializedLabelConfig{
		Labels: []labelConfigInterface{
			&simpleLabelConfig{Name: "simple-label"},
			&simpleLabelConfig{Name: "simple-label-but-deprecated", Deprecated: true, DeprecationMessage: "I'm sure there is a reason"},
			&compoundLabelConfig{
				Prefix: "impact",
				Inner: []labelConfigInterface{
					&simpleLabelConfig{Name: "vyper-sdd"},
					&simpleLabelConfig{Name: "dep", Deprecated: true, DeprecationMessage: "There's no reason"},
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
