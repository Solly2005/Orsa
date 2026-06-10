package workflow

import (
	"strings"
)

var RedFlags = []string{
	"chest pain",
	"shortness of breath",
	"can't breathe",
	"cant breathe",
	"stroke",
	"face droop",
	"severe bleeding",
	"unconscious",
	"suicidal",
	"overdose",
	"anaphylaxis",
}

var negationCues = []string{
	"no ", "not ", "without ", "denies ", "deny ", "denied ", "negative for ",
	"never ", "free of ", "no history of ", "n't ",
}

// DetectRedFlags scans affirmed clinical text (message, chief complaint,
// symptoms) and skips explicitly negated mentions, rather than substring-matching
// the entire serialized state (which flagged denied symptoms like the modifier
// "denies chest pain").
func DetectRedFlags(text string, state State) []string {
	sources := make([]string, 0, len(state.Symptoms)+2)
	sources = append(sources, text, state.ChiefComplaint)
	sources = append(sources, state.Symptoms...)

	seen := map[string]struct{}{}
	matches := make([]string, 0)
	for _, src := range sources {
		for _, flag := range RedFlags {
			if _, ok := seen[flag]; ok {
				continue
			}
			if containsAffirmed(src, flag) {
				matches = append(matches, flag)
				seen[flag] = struct{}{}
			}
		}
	}
	return matches
}

func containsAffirmed(src, phrase string) bool {
	s := strings.ToLower(src)
	p := strings.ToLower(phrase)
	for from := 0; from < len(s); {
		i := strings.Index(s[from:], p)
		if i < 0 {
			return false
		}
		pos := from + i
		start := pos - 24
		if start < 0 {
			start = 0
		}
		if !hasNegationCue(s[start:pos]) {
			return true
		}
		from = pos + len(p)
	}
	return false
}

func hasNegationCue(prefix string) bool {
	for _, cue := range negationCues {
		if strings.Contains(prefix, cue) {
			return true
		}
	}
	return false
}

func DangerZoneVitals(vitals Vitals, _ *float64) *int {
	level := 2
	if vitals.SpO2 != nil && *vitals.SpO2 < 92 {
		return &level
	}
	if vitals.RR != nil && (*vitals.RR > 28 || *vitals.RR < 8) {
		return &level
	}
	if vitals.HR != nil && (*vitals.HR > 130 || *vitals.HR < 40) {
		return &level
	}
	return nil
}
