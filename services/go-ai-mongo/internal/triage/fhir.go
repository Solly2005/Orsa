package triage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"orsa.ai/go-ai-mongo/internal/umls"
)

// buildFHIRBundle emits a FHIR R5 collection Bundle capturing one triage turn.
// SNOMED-coded entries are included when UMLS concepts are provided.
func buildFHIRBundle(state *State, finalESI int, rationale string, concepts []umls.Concept, likelyCondition any, specialty string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	pid := "urn:uuid:anonymous-patient"

	entries := []map[string]any{
		patientEntry(pid),
		triageObservation(now, pid, state, finalESI, rationale, specialty),
	}

	// Condition entries for each UMLS concept.
	for _, con := range concepts {
		entries = append(entries, conditionEntry(now, pid, con))
	}

	// Likely condition as Condition resource (without SNOMED code if UMLS unavailable).
	if likelyCondition != nil {
		if txt, ok := likelyCondition.(string); ok && strings.TrimSpace(txt) != "" {
			entries = append(entries, likelyConditionEntry(now, pid, txt))
		}
	}

	// Red-flag Flag resources.
	for _, rf := range state.RedFlags {
		entries = append(entries, redFlagEntry(now, pid, rf))
	}

	// Vital sign Observations.
	for _, obs := range vitalObservations(now, pid, state.Vitals) {
		entries = append(entries, obs)
	}

	bundle := map[string]any{
		"resourceType": "Bundle",
		"id":           newBundleID(),
		"meta": map[string]any{
			"profile": []string{"http://hl7.org/fhir/StructureDefinition/Bundle"},
		},
		"type":      "collection",
		"timestamp": now,
		"entry":     entries,
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

// ─── resource builders ────────────────────────────────────────────────────────

func patientEntry(pid string) map[string]any {
	return map[string]any{
		"fullUrl": pid,
		"resource": map[string]any{
			"resourceType": "Patient",
			"id":           "anonymous-patient",
			"meta": map[string]any{
				"security": []map[string]any{{
					"system": "http://terminology.hl7.org/CodeSystem/v3-Confidentiality",
					"code":   "R",
					"display": "Restricted",
				}},
			},
		},
	}
}

func triageObservation(now, pid string, state *State, finalESI int, rationale, specialty string) map[string]any {
	components := []map[string]any{
		{
			"code": loinc("46241-6", "Chief complaint - Reported"),
			"valueString": state.ChiefComplaint,
		},
		{
			"code": map[string]any{
				"text": "Emergency Severity Index (ESI)",
			},
			"valueInteger": finalESI,
		},
		{
			"code":        map[string]any{"text": "Triage rationale"},
			"valueString": rationale,
		},
		{
			"code":        map[string]any{"text": "Recommended specialty"},
			"valueString": specialty,
		},
	}
	if len(state.RedFlags) > 0 {
		components = append(components, map[string]any{
			"code":        map[string]any{"text": "Detected red flags"},
			"valueString": joinStrings(state.RedFlags),
		})
	}
	if len(state.Symptoms) > 0 {
		components = append(components, map[string]any{
			"code":        loinc("75325-1", "Symptom"),
			"valueString": joinStrings(state.Symptoms),
		})
	}

	return fhirEntry(map[string]any{
		"resourceType": "Observation",
		"status":       "preliminary",
		"category": []map[string]any{{
			"coding": []map[string]any{{
				"system":  "http://terminology.hl7.org/CodeSystem/observation-category",
				"code":    "survey",
				"display": "Survey",
			}},
		}},
		"code": map[string]any{
			"coding": []map[string]any{{
				"system":  "http://loinc.org",
				"code":    "11283-9",
				"display": "Emergency department triage note",
			}},
			"text": "ORSA AI triage assessment",
		},
		"subject":           subjectRef(pid),
		"effectiveDateTime": now,
		"component":         components,
	})
}

func conditionEntry(now, pid string, con umls.Concept) map[string]any {
	coding := []map[string]any{{
		"system":  "http://snomed.info/sct",
		"code":    con.SnomedCode,
		"display": con.Name,
	}}
	if con.CUI != "" {
		coding = append(coding, map[string]any{
			"system":  "https://uts.nlm.nih.gov/uts/umls",
			"code":    con.CUI,
			"display": con.Name,
		})
	}
	return fhirEntry(map[string]any{
		"resourceType": "Condition",
		"clinicalStatus": map[string]any{
			"coding": []map[string]any{{
				"system":  "http://terminology.hl7.org/CodeSystem/condition-clinical",
				"code":    "active",
				"display": "Active",
			}},
		},
		"verificationStatus": map[string]any{
			"coding": []map[string]any{{
				"system":  "http://terminology.hl7.org/CodeSystem/condition-ver-status",
				"code":    "unconfirmed",
				"display": "Unconfirmed",
			}},
		},
		"code":              map[string]any{"coding": coding, "text": con.Name},
		"subject":           subjectRef(pid),
		"recordedDate":      now,
	})
}

func likelyConditionEntry(now, pid, name string) map[string]any {
	return fhirEntry(map[string]any{
		"resourceType": "Condition",
		"clinicalStatus": map[string]any{
			"coding": []map[string]any{{
				"system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
				"code":   "active",
			}},
		},
		"verificationStatus": map[string]any{
			"coding": []map[string]any{{
				"system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
				"code":   "unconfirmed",
			}},
		},
		"code":         map[string]any{"text": name},
		"subject":      subjectRef(pid),
		"recordedDate": now,
		"note": []map[string]any{{"text": "Most likely condition per AI triage; unconfirmed."}},
	})
}

func redFlagEntry(now, pid, flag string) map[string]any {
	return fhirEntry(map[string]any{
		"resourceType": "Flag",
		"status":       "active",
		"category": []map[string]any{{
			"coding": []map[string]any{{
				"system":  "http://terminology.hl7.org/CodeSystem/flag-category",
				"code":    "clinical",
				"display": "Clinical",
			}},
		}},
		"code":    map[string]any{"text": "RED FLAG: " + flag},
		"subject": subjectRef(pid),
		"period":  map[string]any{"start": now},
	})
}

func vitalObservations(now, pid string, v Vitals) []map[string]any {
	var obs []map[string]any
	if v.HR != nil {
		obs = append(obs, vitalObs(now, pid, "8867-4", "Heart rate", fmt.Sprintf("%.0f", *v.HR), "/min"))
	}
	if v.RR != nil {
		obs = append(obs, vitalObs(now, pid, "9279-1", "Respiratory rate", fmt.Sprintf("%.0f", *v.RR), "/min"))
	}
	if v.SpO2 != nil {
		obs = append(obs, vitalObs(now, pid, "2708-6", "Oxygen saturation", fmt.Sprintf("%.1f", *v.SpO2), "%"))
	}
	if v.Temp != nil {
		obs = append(obs, vitalObs(now, pid, "8310-5", "Body temperature", fmt.Sprintf("%.1f", *v.Temp), "Cel"))
	}
	if v.BP != nil && strings.TrimSpace(*v.BP) != "" {
		obs = append(obs, vitalObs(now, pid, "55284-4", "Blood pressure", *v.BP, "mmHg"))
	}
	return obs
}

func vitalObs(now, pid, loincCode, display, value, unit string) map[string]any {
	return fhirEntry(map[string]any{
		"resourceType": "Observation",
		"status":       "preliminary",
		"category": []map[string]any{{
			"coding": []map[string]any{{
				"system": "http://terminology.hl7.org/CodeSystem/observation-category",
				"code":   "vital-signs",
			}},
		}},
		"code":              loinc(loincCode, display),
		"subject":           subjectRef(pid),
		"effectiveDateTime": now,
		"valueString":       value + " " + unit,
	})
}

// ─── small helpers ────────────────────────────────────────────────────────────

func fhirEntry(resource map[string]any) map[string]any {
	return map[string]any{"resource": resource}
}

func subjectRef(pid string) map[string]any {
	return map[string]any{"reference": pid}
}

func loinc(code, display string) map[string]any {
	return map[string]any{
		"coding": []map[string]any{{
			"system":  "http://loinc.org",
			"code":    code,
			"display": display,
		}},
		"text": display,
	}
}

func joinStrings(items []string) string {
	return strings.Join(items, ", ")
}

func newBundleID() string {
	return fmt.Sprintf("orsa-triage-%d", time.Now().UnixNano())
}
