/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package amend

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/google/uuid"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/tidwall/gjson"
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

	declaresVersions := fileDeclaresVersions(dpFileName)

	var eventSpecifications []model.EventSpecCanonical
	for _, spec := range esNames {
		eventSpec := model.EventSpecCanonical{
			ResourceName: uuid.NewString(),
			Name:         spec,
			Entities:     model.EntitiesDef{},
		}
		if declaresVersions {
			version := 0
			eventSpec.Version = &version
		}
		eventSpecifications = append(eventSpecifications, eventSpec)
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

func fileDeclaresVersions(dpFileName string) bool {
	resources, err := util.MaybeResourcesfromPaths([]string{dpFileName})
	if err != nil {
		return false
	}
	for _, maybeDp := range resources {
		data, _ := maybeDp["data"].(map[string]any)
		specs, _ := data["eventSpecifications"].([]any)
		for _, spec := range specs {
			if specMap, ok := spec.(map[string]any); ok {
				if _, hasVersion := specMap["version"]; hasVersion {
					return true
				}
			}
		}
	}
	return false
}

func AddEventSpecs(ess []model.EventSpecCanonical, file []byte, dpFileName string) ([]byte, error) {
	var output []byte
	var err error
	switch {
	case strings.HasSuffix(dpFileName, ".yaml") || strings.HasSuffix(dpFileName, "yml"):
		output, err = modifyYaml(ess, file)
		if err != nil {
			return []byte{}, err
		}
	case strings.HasSuffix(dpFileName, ".json"):
		output, err = modifyJson(ess, file)
		if err != nil {
			return []byte{}, err
		}
	default:
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

	arrayNode, err := eventSpecificationsNode(node)
	if err != nil {
		return []byte{}, err
	}

	// if array is empty and is represented by [], switch to block style, and alight it back
	if len(arrayNode.Values) == 0 {
		if arrayNode.IsFlowStyle {
			arrayNode.SetIsFlowStyle(false)
			arrayNode.Start.Position.Column = 9
		}
	}

	for _, eventSpec := range ess {
		if err := appendEventSpecToYamlArray(arrayNode, eventSpec); err != nil {
			return []byte{}, err
		}
	}

	// marshal back to YAML with preserved formatting and comments
	output, err := yaml.MarshalWithOptions(node, yaml.WithComment(comments), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return output, nil
}

func appendEventSpecToYamlArray(arrayNode *ast.SequenceNode, eventSpec model.EventSpecCanonical) error {
	// marshal the event spec using gopkg.in/yaml.v3 to use custom marshaler
	specBytes, err := googleyaml.Marshal(eventSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal event spec: %w", err)
	}

	var specNode ast.Node
	if err := yaml.Unmarshal(specBytes, &specNode); err != nil {
		return fmt.Errorf("failed to parse event spec YAML: %w", err)
	}

	if mappingNode, ok := specNode.(*ast.MappingNode); ok {
		mappingNode.SetIsFlowStyle(arrayNode.IsFlowStyle)
	}

	arrayNode.Values = append(arrayNode.Values, specNode)

	return nil
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
	switch {
	case strings.HasSuffix(dpFileName, ".yaml") || strings.HasSuffix(dpFileName, "yml"):
		output, err = setVersionsYaml(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	case strings.HasSuffix(dpFileName, ".json"):
		output, err = setVersionsJson(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("file has not recognized extension %s. Recognized are .yaml, .yml, .json", dpFileName)
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
		return nil, fmt.Errorf("failed to parse data product JSON: %w", err)
	}
	for idx := range versionsByIndex {
		if idx < 0 || idx >= len(dp.Data.EventSpecifications) {
			return nil, fmt.Errorf("version of event specification %d not found", idx)
		}
		if _, ok := dp.Data.EventSpecifications[idx]["version"]; !ok {
			return nil, fmt.Errorf("version of event specification %d not found", idx)
		}
	}

	updatedDp := string(file)

	for idx, version := range versionsByIndex {
		updated, err := sjson.Set(updatedDp, fmt.Sprintf("data.eventSpecifications.%d.version", idx), version)
		if err != nil {
			return nil, err
		}
		updatedDp = updated
	}

	return []byte(updatedDp), nil
}

func setVersionsYaml(versionsByIndex map[int]int, file []byte) ([]byte, error) {
	comments := yaml.CommentMap{}
	var node ast.Node
	if err := yaml.UnmarshalWithOptions(file, &node, yaml.UseOrderedMap(), yaml.CommentToMap(comments)); err != nil {
		return nil, fmt.Errorf("failed to parse data product YAML: %w", err)
	}

	arrayNode, err := eventSpecificationsNode(node)
	if err != nil {
		return nil, err
	}

	for idx, version := range versionsByIndex {
		if idx < 0 || idx >= len(arrayNode.Values) {
			return nil, fmt.Errorf("version of event specification %d not found", idx)
		}

		values := mappingValuesOf(arrayNode.Values[idx])
		versionIdx := slices.IndexFunc(values, func(v *ast.MappingValueNode) bool { return v.Key.String() == "version" })
		if versionIdx < 0 {
			return nil, fmt.Errorf("version of event specification %d not found", idx)
		}

		versionNode, err := yaml.ValueToNode(version)
		if err != nil {
			return nil, fmt.Errorf("failed to build version of event specification %d: %w", idx, err)
		}
		values[versionIdx].Value = versionNode
	}

	output, err := yaml.MarshalWithOptions(node, yaml.WithComment(comments), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return withLineEndingsOf(file, output), nil
}

func mappingValuesOf(node ast.Node) []*ast.MappingValueNode {
	switch n := node.(type) {
	case *ast.MappingNode:
		return n.Values
	case *ast.MappingValueNode:
		return []*ast.MappingValueNode{n}
	default:
		return nil
	}
}

func AddEventSpecVersionsInFile(versionsByIndex map[int]int, dpFileName string) error {
	file, err := os.ReadFile(dpFileName)
	if err != nil {
		return err
	}

	output, err := AddEventSpecVersions(versionsByIndex, file, dpFileName)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dpFileName, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func AddEventSpecVersions(versionsByIndex map[int]int, file []byte, dpFileName string) ([]byte, error) {
	var output []byte
	var err error
	switch {
	case strings.HasSuffix(dpFileName, ".yaml") || strings.HasSuffix(dpFileName, "yml"):
		output, err = addVersionsYaml(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	case strings.HasSuffix(dpFileName, ".json"):
		output, err = addVersionsJson(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("file has not recognized extension %s. Recognized are .yaml, .yml, .json", dpFileName)
	}
	return output, nil
}

// The version goes right after the name, where dp download writes it, and the document is indented the way dp download writes it.
func addVersionsJson(versionsByIndex map[int]int, file []byte) ([]byte, error) {
	content := string(file)

	if err := checkJsonSpecsExistWithoutVersion(content, versionsByIndex); err != nil {
		return nil, err
	}

	for idx, version := range versionsByIndex {
		path := fmt.Sprintf("data.eventSpecifications.%d", idx)

		updated, err := sjson.SetRaw(content, path, jsonSpecWithVersionAfterName(gjson.Get(content, path), version))
		if err != nil {
			return nil, err
		}
		content = updated
	}

	output, err := indentJsonKeepingFinalNewline(content, file)
	if err != nil {
		return nil, err
	}

	return withLineEndingsOf(file, output), nil
}

func checkJsonSpecsExistWithoutVersion(content string, versionsByIndex map[int]int) error {
	count := gjson.Get(content, "data.eventSpecifications.#").Int()

	for idx := range versionsByIndex {
		if idx < 0 || int64(idx) >= count {
			return fmt.Errorf("event specification %d not found", idx)
		}
		if gjson.Get(content, fmt.Sprintf("data.eventSpecifications.%d.version", idx)).Exists() {
			return fmt.Errorf("event specification %d already declares a version", idx)
		}
	}

	return nil
}

func jsonSpecWithVersionAfterName(spec gjson.Result, version int) string {
	versionMember := fmt.Sprintf(`"version":%d`, version)

	var members []string
	added := false
	spec.ForEach(func(key, value gjson.Result) bool {
		members = append(members, key.Raw+":"+value.Raw)
		if key.String() == "name" {
			members = append(members, versionMember)
			added = true
		}
		return true
	})
	if !added {
		members = append(members, versionMember)
	}

	return "{" + strings.Join(members, ",") + "}"
}

func indentJsonKeepingFinalNewline(content string, file []byte) ([]byte, error) {
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, []byte(content)); err != nil {
		return nil, fmt.Errorf("failed to compact data product JSON: %w", err)
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, compacted.Bytes(), "", "  "); err != nil {
		return nil, fmt.Errorf("failed to indent data product JSON: %w", err)
	}
	output := indented.Bytes()
	if bytes.HasSuffix(file, []byte("\n")) {
		output = append(output, '\n')
	}

	return output, nil
}

// The version goes right after the name, where dp download writes it.
func addVersionsYaml(versionsByIndex map[int]int, file []byte) ([]byte, error) {
	comments := yaml.CommentMap{}
	var node ast.Node
	if err := yaml.UnmarshalWithOptions(file, &node, yaml.UseOrderedMap(), yaml.CommentToMap(comments)); err != nil {
		return nil, fmt.Errorf("failed to parse data product YAML: %w", err)
	}

	arrayNode, err := eventSpecificationsNode(node)
	if err != nil {
		return nil, err
	}

	for idx, version := range versionsByIndex {
		specNode, err := eventSpecMappingNode(arrayNode, idx)
		if err != nil {
			return nil, err
		}

		nameIdx, err := nameIndexOfSpecWithoutVersion(specNode, idx)
		if err != nil {
			return nil, err
		}

		indent := strings.Repeat(" ", specNode.Values[nameIdx].Key.GetToken().Position.Column-1)
		versionValue, err := versionMappingValueNode(indent, version, idx)
		if err != nil {
			return nil, err
		}

		specNode.Values = slices.Insert(specNode.Values, nameIdx+1, versionValue)
		arrayNode.Values[idx] = specNode
	}

	output, err := yaml.MarshalWithOptions(node, yaml.WithComment(comments), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return withLineEndingsOf(file, output), nil
}

func eventSpecMappingNode(arrayNode *ast.SequenceNode, idx int) (*ast.MappingNode, error) {
	if idx < 0 || idx >= len(arrayNode.Values) {
		return nil, fmt.Errorf("event specification %d not found", idx)
	}

	switch n := arrayNode.Values[idx].(type) {
	case *ast.MappingNode:
		return n, nil
	case *ast.MappingValueNode:
		return ast.Mapping(n.GetToken(), n.IsFlowStyle, n), nil
	default:
		return nil, fmt.Errorf("event specification %d is not a mapping", idx)
	}
}

func nameIndexOfSpecWithoutVersion(specNode *ast.MappingNode, idx int) (int, error) {
	nameIdx := -1
	for i, v := range specNode.Values {
		switch v.Key.String() {
		case "version":
			return 0, fmt.Errorf("event specification %d already declares a version", idx)
		case "name":
			nameIdx = i
		}
	}
	if nameIdx < 0 {
		return 0, fmt.Errorf("name of event specification %d not found", idx)
	}

	return nameIdx, nil
}

func versionMappingValueNode(indent string, version int, idx int) (*ast.MappingValueNode, error) {
	var versionNode ast.Node
	if err := yaml.Unmarshal([]byte(fmt.Sprintf("%sversion: %d", indent, version)), &versionNode); err != nil {
		return nil, fmt.Errorf("failed to parse version YAML: %w", err)
	}
	versionMapping, ok := versionNode.(*ast.MappingNode)
	if !ok || len(versionMapping.Values) != 1 {
		return nil, fmt.Errorf("failed to build version of event specification %d", idx)
	}

	return versionMapping.Values[0], nil
}

func eventSpecificationsNode(node ast.Node) (*ast.SequenceNode, error) {
	mappingNode, ok := node.(*ast.MappingNode)
	if !ok {
		return nil, errors.New("root node is not a mapping")
	}

	var dataValue ast.Node
	for _, value := range mappingNode.Values {
		if value.Key.String() == "data" {
			dataValue = value.Value
			break
		}
	}
	if dataValue == nil {
		return nil, errors.New("'data' key not found")
	}

	dataMappingNode, ok := dataValue.(*ast.MappingNode)
	if !ok {
		return nil, errors.New("data value is not a mapping")
	}

	var eventSpecsValue ast.Node
	for _, value := range dataMappingNode.Values {
		if value.Key.String() == "eventSpecifications" {
			eventSpecsValue = value.Value
			break
		}
	}
	if eventSpecsValue == nil {
		return nil, errors.New("'eventSpecifications' key not found")
	}

	arrayNode, ok := eventSpecsValue.(*ast.SequenceNode)
	if !ok {
		return nil, errors.New("data.eventSpecifications is not an array")
	}

	return arrayNode, nil
}

func withLineEndingsOf(file []byte, output []byte) []byte {
	if !bytes.Contains(file, []byte("\r\n")) {
		return output
	}
	return bytes.ReplaceAll(output, []byte("\n"), []byte("\r\n"))
}
