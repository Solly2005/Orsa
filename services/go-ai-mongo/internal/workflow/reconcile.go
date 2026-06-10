package workflow

import "math"

type ReconcileResult struct {
	FinalESI            int  `json:"final_esi"`
	BertESI             int  `json:"bert_esi"`
	GptESI              int  `json:"gpt_esi"`
	RedFlagFloor        *int `json:"red_flag_floor"`
	Disagreement        bool `json:"disagreement"`
	GptWantedDeescalate bool `json:"gpt_wanted_deescalate"`
}

func Reconcile(bertLevel int, gptLevel int, redFlagFloor *int) ReconcileResult {
	final := min(bertLevel, gptLevel)
	if redFlagFloor != nil {
		final = min(final, *redFlagFloor)
	}

	return ReconcileResult{
		FinalESI:            final,
		BertESI:             bertLevel,
		GptESI:              gptLevel,
		RedFlagFloor:        redFlagFloor,
		Disagreement:        int(math.Abs(float64(bertLevel-gptLevel))) >= 2,
		GptWantedDeescalate: gptLevel > bertLevel,
	}
}
