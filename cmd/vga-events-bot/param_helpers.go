package main

import (
	"fmt"
	"strings"
)

// extractQuotedParam extracts and validates a multi-word parameter
// Used by /search, /near, /note, and filter subcommands
func extractQuotedParam(parts []string, minParts, maxLength int, fieldName string) (string, error) {
	if len(parts) < minParts {
		return "", fmt.Errorf("❌ Please provide %s.", fieldName)
	}

	param := strings.Join(parts[1:], " ")
	param = strings.Trim(param, `"'`)

	param, errMsg := validateUserInput(param, maxLength, fieldName)
	if errMsg != "" {
		return "", fmt.Errorf("%s", errMsg)
	}

	return param, nil
}
