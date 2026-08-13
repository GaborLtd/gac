package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Input 是 prompt template 可使用的資料。
type Input struct {
	Language string
	Stat     string
	Diff     string
	Context  string
}

const defaultTemplate = `Based on the following Git diff, write one commit message that follows the Conventional Commits format.
Output only the message itself, without explanation or code fences. Write the message in {{.Language}}.

=== Changed files ===
{{.Stat}}

=== Diff content ===
{{.Diff}}

=== Additional context ===
{{.Context}}`

// Build 建立 prompt；自訂 template 仍會附加固定輸出契約。
func Build(in Input, custom string) (string, error) {
	tplText := custom
	if tplText == "" {
		tplText = defaultTemplate
	}
	tpl, err := template.New("prompt").Option("missingkey=error").Parse(tplText)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	if err := tpl.Execute(&body, in); err != nil {
		return "", err
	}
	language := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(in.Language, "\r", " "), "\n", " "))
	if language == "" {
		language = "en"
	}
	return fmt.Sprintf("%s\n\nFixed output contract: return exactly one Conventional Commits message only. Write the message in %s. Do not include explanations or code fences.", body.String(), language), nil
}
