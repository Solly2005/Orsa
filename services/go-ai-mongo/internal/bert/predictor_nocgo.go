//go:build !cgo

package bert

import (
	"fmt"

	"orsa.ai/go-ai-mongo/internal/config"
)

// Predictor is unavailable when the service is built without CGo. Keeping this
// stub lets the service compile and use the safe non-BERT triage fallback.
type Predictor struct{}

// New reports the missing build prerequisite. The caller treats this as an
// unavailable specialist signal and continues with the safe fallback path.
func New(config.Config) (*Predictor, error) {
	return nil, fmt.Errorf("BERT-ESI ONNX requires CGO_ENABLED=1 and a C compiler")
}

func (p *Predictor) Predict(string) (level int, confidence float64) {
	return 5, 0
}

func (p *Predictor) Close() {}
