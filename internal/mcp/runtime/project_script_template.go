package runtime

import _ "embed"

//go:embed templates/circular-mcp.sh
var projectScriptTemplate []byte
