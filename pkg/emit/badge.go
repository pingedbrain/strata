package emit

import (
	"encoding/json"
	"fmt"
)

// Badge renders a shields.io endpoint JSON payload for requirement
// coverage — drop it behind an endpoint or commit it as a badge.json
// and reference shields.io/endpoint?url=…
// @spec emit/badge
func Badge(label string, covered, total int) []byte {
	pct := 0
	if total > 0 {
		pct = covered * 100 / total
	}
	color := "red"
	switch {
	case pct >= 90:
		color = "brightgreen"
	case pct >= 75:
		color = "green"
	case pct >= 50:
		color = "yellow"
	case pct > 0:
		color = "orange"
	}
	payload := map[string]any{
		"schemaVersion": 1,
		"label":         label,
		"message":       fmt.Sprintf("%d%% (%d/%d)", pct, covered, total),
		"color":         color,
	}
	b, _ := json.Marshal(payload)
	return append(b, '\n')
}
