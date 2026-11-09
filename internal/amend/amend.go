package amend

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/google/uuid"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/tidwall/sjson"
	googleyaml "gopkg.in/yaml.v3"
)

func PrintEventSpecsToStdout(esNames []string, ext string) error {
	var bytes []byte
	var err error

	var eventSpecifications []model.EventSpecCanonical
	for _, spec := range esNames {
		eventSpecifications = append(eventSpecifications, model.EventSpecCanonical{
			ResourceName: uuid.NewString(),
			Name:         spec,
			Entities:     model.EntitiesDef{},
		})
	}

	if ext == "yaml" {
		bytes, err = yaml.Marshal(eventSpecifications)
		if err != nil {
			return err
		}
	} else {
		bytes, err = json.MarshalIndent(eventSpecifications, "", "  ")
		if err != nil {
			return err
		}
	}

	_, err = fmt.Println(string(bytes))
	return err
}

func AddEventSpecsToFile(esNames []string, dpFileName string) error {

	if _, err := os.Stat(dpFileName); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist at %s", dpFileName)
	}

	var eventSpecifications []model.EventSpecCanonical
	for _, spec := range esNames {
		eventSpecifications = append(eventSpecifications, model.EventSpecCanonical{
			ResourceName: uuid.NewString(),
			Name:         spec,
			Entities:     model.EntitiesDef{},
		})
	}

	file, err := os.ReadFile(dpFileName)
	if err != nil {
		return err
	}

	output, err := AddEventSpecs(eventSpecifications, file, dpFileName)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dpFileName, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func AddEventSpecs(ess []model.EventSpecCanonical, file []byte, dpFileName string) ([]byte, error) {
	var output []byte
	var err error
	if strings.HasSuffix(dpFileName, ".yaml") || strings.HasSuffix(dpFileName, "yml") {
		output, err = modifyYaml(ess, file)
		if err != nil {
			return []byte{}, err
		}
	} else if strings.HasSuffix(dpFileName, ".json") {
		output, err = modifyJson(ess, file)
		if err != nil {
			return []byte{}, err
		}
	} else {
		return []byte{}, fmt.Errorf("file has not recognized extension %s. Recognized are .yaml, .yml, .json", dpFileName)
	}
	return output, nil
}

func modifyJson(ess []model.EventSpecCanonical, file []byte) ([]byte, error) {

	updatedDp := string(file)

	for _, eventSpec := range ess {
		updated, err := sjson.Set(updatedDp, "data.eventSpecifications.-1", eventSpec)
		if err != nil {
			return []byte{}, err
		}
		updatedDp = updated
	}

	return []byte(updatedDp), nil
}

func modifyYaml(ess []model.EventSpecCanonical, file []byte) ([]byte, error) {
	comments := yaml.CommentMap{}
	var node ast.Node
	if err := yaml.UnmarshalWithOptions(file, &node, yaml.UseOrderedMap(), yaml.CommentToMap(comments)); err != nil {
		return []byte{}, fmt.Errorf("failed to parse data product YAML: %w", err)
	}

	mappingNode, ok := node.(*ast.MappingNode)
	if !ok {
		return []byte{}, fmt.Errorf("root node is not a mapping")
	}

	var dataValue ast.Node
	for _, value := range mappingNode.Values {
		mapValue := value
		if mapValue.Key.String() == "data" {
			dataValue = mapValue.Value
			break
		}
	}
	if dataValue == nil {
		return []byte{}, fmt.Errorf("'data' key not found")
	}

	dataMappingNode, ok := dataValue.(*ast.MappingNode)
	if !ok {
		return []byte{}, fmt.Errorf("data value is not a mapping")
	}

	var eventSpecsValue ast.Node
	for _, value := range dataMappingNode.Values {
		mapValue := value
		if mapValue.Key.String() == "eventSpecifications" {
			eventSpecsValue = mapValue.Value
			break
		}
	}
	if eventSpecsValue == nil {
		return []byte{}, fmt.Errorf("'eventSpecifications' key not found")
	}

	arrayNode, ok := eventSpecsValue.(*ast.SequenceNode)
	if !ok {
		return []byte{}, fmt.Errorf("data.eventSpecifications is not an array")
	}

	// if array is empty and is represented by [], switch to block style, and alight it back
	if len(arrayNode.Values) == 0 {
		if arrayNode.IsFlowStyle {
			arrayNode.SetIsFlowStyle(false)
			arrayNode.Start.Position.Column = 9
		}
	}

	for _, eventSpec := range ess {
		// marshal the event spec using gopkg.in/yaml.v3 to use custom marshaler
		specBytes, err := googleyaml.Marshal(eventSpec)
		if err != nil {
			return []byte{}, fmt.Errorf("failed to marshal event spec: %w", err)
		}

		var specNode ast.Node
		if err := yaml.Unmarshal(specBytes, &specNode); err != nil {
			return []byte{}, fmt.Errorf("failed to parse event spec YAML: %w", err)
		}

		if mappingNode, ok := specNode.(*ast.MappingNode); ok {
			mappingNode.SetIsFlowStyle(arrayNode.IsFlowStyle)
		}

		arrayNode.Values = append(arrayNode.Values, specNode)
	}

	// marshal back to YAML with preserved formatting and comments
	output, err := yaml.MarshalWithOptions(node, yaml.WithComment(comments), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return output, nil

}

func SetEventSpecVersionsInFile(versionsByIndex map[int]int, dpFileName string) error {
	file, err := os.ReadFile(dpFileName)
	if err != nil {
		return err
	}

	output, err := SetEventSpecVersions(versionsByIndex, file, dpFileName)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dpFileName, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func SetEventSpecVersions(versionsByIndex map[int]int, file []byte, dpFileName string) ([]byte, error) {
	var output []byte
	var err error
	if strings.HasSuffix(dpFileName, ".yaml") || strings.HasSuffix(dpFileName, "yml") {
		output, err = setVersionsYaml(versionsByIndex, file)
		if err != nil {
			return []byte{}, err
		}
	} else if strings.HasSuffix(dpFileName, ".json") {
		output, err = setVersionsJson(versionsByIndex, file)
		if err != nil {
			return []byte{}, err
		}
	} else {
		return []byte{}, fmt.Errorf("file has not recognized extension %s. Recognized are .yaml, .yml, .json", dpFileName)
	}
	return output, nil
}

func setVersionsJson(versionsByIndex map[int]int, file []byte) ([]byte, error) {

	var dp struct {
		Data struct {
			EventSpecifications []map[string]json.RawMessage `json:"eventSpecifications"`
		} `json:"data"`
	}
	if err := json.Unmarshal(file, &dp); err != nil {
		return []byte{}, fmt.Errorf("failed to parse data product JSON: %w", err)
	}
	for idx := range versionsByIndex {
		if idx < 0 || idx >= len(dp.Data.EventSpecifications) {
			return []byte{}, fmt.Errorf("version of event specification %d not found", idx)
		}
		if _, ok := dp.Data.EventSpecifications[idx]["version"]; !ok {
			return []byte{}, fmt.Errorf("version of event specification %d not found", idx)
		}
	}

	updatedDp := string(file)

	for idx, version := range versionsByIndex {
		updated, err := sjson.Set(updatedDp, fmt.Sprintf("data.eventSpecifications.%d.version", idx), version)
		if err != nil {
			return []byte{}, err
		}
		updatedDp = updated
	}

	return []byte(updatedDp), nil
}

func setVersionsYaml(versionsByIndex map[int]int, file []byte) ([]byte, error) {
	parsed, err := parser.ParseBytes(file, parser.ParseComments)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to parse data product YAML: %w", err)
	}

	type replacement struct {
		offset int
		old    string
		new    string
	}

	runes := []rune(string(file))

	lineStarts := []int{0}
	for i, r := range runes {
		if r == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	var replacements []replacement
	for idx, version := range versionsByIndex {
		path, err := yaml.PathString(fmt.Sprintf("$.data.eventSpecifications[%d].version", idx))
		if err != nil {
			return []byte{}, err
		}
		node, err := path.FilterFile(parsed)
		if err != nil {
			return []byte{}, fmt.Errorf("version of event specification %d not found: %w", idx, err)
		}
		tk := node.GetToken()
		line := tk.Position.Line - 1
		if line < 0 || line >= len(lineStarts) {
			return []byte{}, fmt.Errorf("version of event specification %d could not be located", idx)
		}
		offset := lineStarts[line] + tk.Position.Column - 1
		oldRunes := []rune(tk.Value)
		if offset < 0 || offset+len(oldRunes) > len(runes) || string(runes[offset:offset+len(oldRunes)]) != tk.Value {
			return []byte{}, fmt.Errorf("version of event specification %d could not be located", idx)
		}
		replacements = append(replacements, replacement{offset: offset, old: tk.Value, new: fmt.Sprintf("%d", version)})
	}

	sort.Slice(replacements, func(i, j int) bool { return replacements[i].offset > replacements[j].offset })

	for _, r := range replacements {
		updated := append([]rune{}, runes[:r.offset]...)
		updated = append(updated, []rune(r.new)...)
		runes = append(updated, runes[r.offset+len([]rune(r.old)):]...)
	}

	return []byte(string(runes)), nil
}
