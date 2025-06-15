package cmd

import (
	"encoding/json"
	"fmt"
)

// printJSON prints data as formatted JSON
func printJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

// getStatusSymbol returns a symbol for the git status
func getStatusSymbol(status string) string {
	switch status {
	case "A":
		return "+"
	case "M":
		return "~"
	case "D":
		return "-"
	case "R":
		return "→"
	case "C":
		return "©"
	case "?":
		return "?"
	default:
		return status
	}
}
