package agent

import "strings"

// userMessageSignalsProfileCorrection reports whether the user's message likely
// corrects how the assistant addressed them or interpreted USER PROFILE text.
// When true, we force an immediate background memory review so misread profile
// entries get replaced even if the foreground model only apologizes.
func userMessageSignalsProfileCorrection(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	if m == "" {
		return false
	}
	signals := []string{
		"my name is",
		"my name's",
		"not my name",
		"that's not my name",
		"that is not my name",
		"call me ",
		"don't call me",
		"do not call me",
		"stop calling me",
		"wrong name",
		"not just h",
		"not the letter",
		"way too literally",
		"misread",
		"you got my name",
		"you have my name wrong",
		"wym hey",
		"bro my name",
		"lowercase h not",
		"name is haider",
	}
	for _, s := range signals {
		if strings.Contains(m, s) {
			return true
		}
	}
	return false
}
