package reconciliation

import (
	"encoding/json"
	"fmt"
)

func mapAnyToByteSlice(m map[string]any) ([]byte, error) {
	asJSON, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %w", err)
	}
	return asJSON, nil
}
