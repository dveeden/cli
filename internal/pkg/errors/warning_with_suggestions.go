package errors

import (
	"fmt"
	"os"
)

const (
	warningsMessageHeader = "Warning: "
	reasonMessageHeader   = "\nReason: "
)

type WarningWithSuggestions struct {
	warnMsg        string
	reasonMsg      string
	suggestionsMsg string
}

func NewWarningWithSuggestions(warnMsg string, reasonMsg string, suggestionsMsg string) *WarningWithSuggestions {
	return &WarningWithSuggestions{
		warnMsg:        warnMsg,
		reasonMsg:      reasonMsg,
		suggestionsMsg: suggestionsMsg,
	}
}

func (w *WarningWithSuggestions) DisplayWarningWithSuggestions() {
	if w.warnMsg != "" && w.reasonMsg != "" && w.suggestionsMsg != "" {
		msg := warningsMessageHeader + w.warnMsg + "\n"
		msg += reasonMessageHeader + w.reasonMsg + "\n"
		msg += ComposeSuggestionsMessage(w.suggestionsMsg) + "\n"

		_, _ = fmt.Fprint(os.Stderr, msg)
	}
}
