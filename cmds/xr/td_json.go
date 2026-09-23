package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func printTDJSON(out io.Writer, tests []*TD) error {
	results := make([]*tdJSON, 0, len(tests))
	for _, td := range tests {
		results = append(results, newTDJSON(td))
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

type tdJSON struct {
	Name    string        `json:"name"`
	Status  string        `json:"status"`
	Pass    int           `json:"pass"`
	Fail    int           `json:"fail"`
	Warn    int           `json:"warn"`
	Skip    int           `json:"skip"`
	Entries []tdJSONEntry `json:"entries"`
}

type tdJSONEntry struct {
	Subtest *tdJSON `json:"subtest,omitempty"`
	Status  string  `json:"status,omitempty"`
	Text    string  `json:"text,omitempty"`
}

func newTDJSON(td *TD) *tdJSON {
	result := &tdJSON{
		Name:    td.TestName,
		Status:  tdJSONStatus(td.Status),
		Pass:    td.NumPass,
		Fail:    td.NumFail,
		Warn:    td.NumWarn,
		Skip:    td.NumSkip,
		Entries: []tdJSONEntry{},
	}

	showLogs := td.Config != nil && td.Config.ShowLogs
	for _, entry := range td.Logs {
		if entry.Type != LOG || showLogs || td.Status == FAIL {
			result.Entries = append(result.Entries, newTDJSONEntry(entry))
		}
	}
	return result
}

func newTDJSONEntry(entry *LogEntry) tdJSONEntry {
	if entry.Subtest != nil {
		return tdJSONEntry{Subtest: newTDJSON(entry.Subtest)}
	}
	return tdJSONEntry{
		Status: tdJSONStatus(entry.Type),
		Text:   entry.Text,
	}
}

func tdJSONStatus(status int) string {
	if status <= 0 || status >= len(StatusText) {
		panic(fmt.Sprintf("invalid TD status: %d", status))
	}
	return StatusText[status]
}
