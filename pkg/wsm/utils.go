package wsm

import (
	"encoding/json"
	"fmt"
)

// printJSON prints data as formatted JSON
func PrintJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

// GetStatusSymbol returns a symbol for the git status
func GetStatusSymbol(status string) string {
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
