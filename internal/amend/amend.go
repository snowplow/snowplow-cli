package amend

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
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
			return nil, err
		}
	} else if strings.HasSuffix(dpFileName, ".json") {
		output, err = setVersionsJson(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	} else {
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
	parsed, err := parser.ParseBytes(file, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data product YAML: %w", err)
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
			return nil, err
		}
		node, err := path.FilterFile(parsed)
		if err != nil {
			return nil, fmt.Errorf("version of event specification %d not found: %w", idx, err)
		}
		tk := node.GetToken()
		line := tk.Position.Line - 1
		if line < 0 || line >= len(lineStarts) {
			return nil, fmt.Errorf("version of event specification %d could not be located", idx)
		}
		offset := lineStarts[line] + tk.Position.Column - 1
		oldRunes := []rune(tk.Value)
		if offset < 0 || offset+len(oldRunes) > len(runes) || string(runes[offset:offset+len(oldRunes)]) != tk.Value {
			return nil, fmt.Errorf("version of event specification %d could not be located", idx)
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
	if strings.HasSuffix(dpFileName, ".yaml") || strings.HasSuffix(dpFileName, "yml") {
		output, err = addVersionsYaml(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	} else if strings.HasSuffix(dpFileName, ".json") {
		output, err = addVersionsJson(versionsByIndex, file)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("file has not recognized extension %s. Recognized are .yaml, .yml, .json", dpFileName)
	}
	return output, nil
}

func isJsonSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// The version goes right after the name, where dp download writes it, copying the spacing of the name member.
func addVersionsJson(versionsByIndex map[int]int, file []byte) ([]byte, error) {
	content := string(file)

	type insertion struct {
		offset int
		text   string
	}

	count := gjson.Get(content, "data.eventSpecifications.#").Int()

	var insertions []insertion
	for idx, version := range versionsByIndex {
		if idx < 0 || int64(idx) >= count {
			return nil, fmt.Errorf("event specification %d not found", idx)
		}
		if gjson.Get(content, fmt.Sprintf("data.eventSpecifications.%d.version", idx)).Exists() {
			return nil, fmt.Errorf("event specification %d already declares a version", idx)
		}
		name := gjson.Get(content, fmt.Sprintf("data.eventSpecifications.%d.name", idx))
		if !name.Exists() || name.Index <= 0 {
			return nil, fmt.Errorf("name of event specification %d could not be located", idx)
		}

		keyEnd := name.Index
		for keyEnd > 0 && isJsonSpace(content[keyEnd-1]) {
			keyEnd--
		}
		if keyEnd == 0 || content[keyEnd-1] != ':' {
			return nil, fmt.Errorf("name of event specification %d could not be located", idx)
		}
		keyEnd--
		for keyEnd > 0 && isJsonSpace(content[keyEnd-1]) {
			keyEnd--
		}
		keyStart := keyEnd - len(`"name"`)
		if keyStart < 0 || content[keyStart:keyEnd] != `"name"` {
			return nil, fmt.Errorf("name of event specification %d could not be located", idx)
		}
		memberStart := keyStart
		for memberStart > 0 && isJsonSpace(content[memberStart-1]) {
			memberStart--
		}
		if memberStart == 0 || (content[memberStart-1] != '{' && content[memberStart-1] != ',') {
			return nil, fmt.Errorf("name of event specification %d could not be located", idx)
		}

		text := fmt.Sprintf(`,%s"version"%s%d`, content[memberStart:keyStart], content[keyEnd:name.Index], version)
		insertions = append(insertions, insertion{offset: name.Index + len(name.Raw), text: text})
	}

	sort.Slice(insertions, func(i, j int) bool { return insertions[i].offset > insertions[j].offset })

	for _, ins := range insertions {
		content = content[:ins.offset] + ins.text + content[ins.offset:]
	}

	return []byte(content), nil
}

// The version goes right after the name, where dp download writes it: on its own line in block style, as the next entry in flow style.
func addVersionsYaml(versionsByIndex map[int]int, file []byte) ([]byte, error) {
	parsed, err := parser.ParseBytes(file, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data product YAML: %w", err)
	}

	type insertion struct {
		offset int
		text   string
	}

	runes := []rune(string(file))

	lineStarts := []int{0}
	for i, r := range runes {
		if r == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	runeOffset := func(pos *token.Position) (int, bool) {
		line := pos.Line - 1
		if line < 0 || line >= len(lineStarts) {
			return 0, false
		}
		offset := lineStarts[line] + pos.Column - 1
		if offset < 0 || offset > len(runes) {
			return 0, false
		}
		return offset, true
	}

	var insertions []insertion
	for idx, version := range versionsByIndex {
		path, err := yaml.PathString(fmt.Sprintf("$.data.eventSpecifications[%d]", idx))
		if err != nil {
			return nil, err
		}
		node, err := path.FilterFile(parsed)
		if err != nil {
			return nil, fmt.Errorf("event specification %d not found: %w", idx, err)
		}

		var values []*ast.MappingValueNode
		isFlowStyle := false
		var flowEnd *token.Token
		switch n := node.(type) {
		case *ast.MappingNode:
			values = n.Values
			isFlowStyle = n.IsFlowStyle
			flowEnd = n.End
		case *ast.MappingValueNode:
			values = []*ast.MappingValueNode{n}
		default:
			return nil, fmt.Errorf("event specification %d is not a mapping", idx)
		}

		nameIdx := -1
		for i, v := range values {
			switch v.Key.GetToken().Value {
			case "version":
				return nil, fmt.Errorf("event specification %d already declares a version", idx)
			case "name":
				nameIdx = i
			}
		}
		if nameIdx < 0 {
			return nil, fmt.Errorf("name of event specification %d not found", idx)
		}

		nameKey := values[nameIdx].Key.GetToken()
		nameValue := values[nameIdx].Value.GetToken()
		if nameValue.Type == token.LiteralType || nameValue.Type == token.FoldedType || nameValue.Position.Line != nameKey.Position.Line || strings.Contains(strings.TrimSpace(nameValue.Origin), "\n") {
			return nil, fmt.Errorf("name of event specification %d spans several lines, add its version by hand", idx)
		}

		if isFlowStyle {
			if nameIdx+1 < len(values) {
				offset, ok := runeOffset(values[nameIdx+1].Key.GetToken().Position)
				if !ok {
					return nil, fmt.Errorf("name of event specification %d could not be located", idx)
				}
				insertions = append(insertions, insertion{offset: offset, text: fmt.Sprintf("version: %d, ", version)})
				continue
			}
			if flowEnd == nil {
				return nil, fmt.Errorf("name of event specification %d could not be located", idx)
			}
			offset, ok := runeOffset(flowEnd.Position)
			if !ok || offset >= len(runes) || runes[offset] != '}' {
				return nil, fmt.Errorf("name of event specification %d could not be located", idx)
			}
			closing := offset
			for offset > 0 && unicode.IsSpace(runes[offset-1]) {
				offset--
			}
			lineStart := offset
			for lineStart > 0 && runes[lineStart-1] != '\n' {
				lineStart--
			}
			if strings.ContainsRune(string(runes[offset:closing]), '\n') && strings.ContainsRune(string(runes[lineStart:offset]), '#') {
				return nil, fmt.Errorf("event specification %d may end in a comment, add its version by hand", idx)
			}
			text := fmt.Sprintf(", version: %d", version)
			if offset > 0 && runes[offset-1] == ',' {
				text = fmt.Sprintf(" version: %d,", version)
			}
			insertions = append(insertions, insertion{offset: offset, text: text})
			continue
		}

		line := nameKey.Position.Line - 1
		if line < 0 || line >= len(lineStarts) {
			return nil, fmt.Errorf("name of event specification %d could not be located", idx)
		}
		lineEnd := len(runes)
		if line+1 < len(lineStarts) {
			lineEnd = lineStarts[line+1] - 1
		}
		newline := "\n"
		if strings.Contains(string(file), "\r\n") {
			newline = "\r\n"
		}
		if lineEnd > 0 && runes[lineEnd-1] == '\r' {
			lineEnd--
		}
		indent := strings.Repeat(" ", nameKey.Position.Column-1)
		insertions = append(insertions, insertion{offset: lineEnd, text: fmt.Sprintf("%s%sversion: %d", newline, indent, version)})
	}

	sort.Slice(insertions, func(i, j int) bool { return insertions[i].offset > insertions[j].offset })

	for _, ins := range insertions {
		updated := append([]rune{}, runes[:ins.offset]...)
		updated = append(updated, []rune(ins.text)...)
		runes = append(updated, runes[ins.offset:]...)
	}

	return []byte(string(runes)), nil
}
