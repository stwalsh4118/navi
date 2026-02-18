package pm

import _ "embed"

var (
	//go:embed templates/system-prompt.md
	SystemPromptTemplate string

	//go:embed templates/output-schema.json
	OutputSchemaTemplate string
)
