//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	key := "dynamicKey"
	value := "dynamicValue"

	payload := map[string]interface{}{
		"staticKey": "staticValue",
		key:         value,
	}

	jsonData, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(jsonData))
}
