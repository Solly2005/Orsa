//go:build cgo

package bert

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
	"orsa.ai/go-ai-mongo/internal/config"
)

var (
	ortOnce sync.Once
	ortErr  error
)

func initORT(libPath string) error {
	ortOnce.Do(func() {
		if libPath != "" {
			ort.SetSharedLibraryPath(libPath)
		}
		ortErr = ort.InitializeEnvironment()
	})
	return ortErr
}

// Predictor runs Bio_ClinicalBERT ESI classification via ONNX Runtime.
// It implements triage.BertPredictor.
type Predictor struct {
	vocab   map[string]int64
	session *ort.DynamicAdvancedSession
}

// New loads the BERT model and vocab from cfg.BertOnnxDir.
// Returns an error (and nil Predictor) if the model or runtime is unavailable;
// the caller should fall back to nil (safe ESI-5 default).
func New(cfg config.Config) (*Predictor, error) {
	if err := initORT(cfg.OnnxRuntimeLib); err != nil {
		return nil, fmt.Errorf("onnxruntime init: %w", err)
	}

	dir := cfg.BertOnnxDir
	// Prefer model.onnx at root; fall back to onnx/model.onnx.
	modelPath := filepath.Join(dir, "model.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		modelPath = filepath.Join(dir, "onnx", "model.onnx")
	}
	vocabPath := filepath.Join(dir, "vocab.txt")

	vocab, err := LoadVocab(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("load vocab %s: %w", vocabPath, err)
	}

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"logits"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create ort session %s: %w", modelPath, err)
	}

	return &Predictor{vocab: vocab, session: session}, nil
}

// Predict tokenizes text, runs BERT inference, and returns the ESI level (1-5)
// and softmax confidence. On any error it returns ESI-5 with 0 confidence so
// the escalate-only reconciler can treat it as the safe neutral signal.
func (p *Predictor) Predict(text string) (level int, confidence float64) {
	shape := ort.NewShape(1, int64(maxSeq))

	inputIDs, attnMask, _ := Tokenize(p.vocab, text)

	idTensor, err := ort.NewTensor(shape, inputIDs)
	if err != nil {
		return 5, 0
	}
	defer idTensor.Destroy()

	amTensor, err := ort.NewTensor(shape, attnMask)
	if err != nil {
		return 5, 0
	}
	defer amTensor.Destroy()

	// token_type_ids: all zeros (single segment) — reuse attention mask shape
	typeIDs := make([]int64, maxSeq)
	tiTensor, err := ort.NewTensor(shape, typeIDs)
	if err != nil {
		return 5, 0
	}
	defer tiTensor.Destroy()

	outputs := []ort.Value{nil} // nil → auto-allocate by ORT
	if err := p.session.Run(
		[]ort.Value{idTensor, amTensor, tiTensor},
		outputs,
	); err != nil {
		return 5, 0
	}
	if outputs[0] == nil {
		return 5, 0
	}
	defer outputs[0].Destroy()

	logitsTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return 5, 0
	}
	logits := logitsTensor.GetData()
	if len(logits) < 5 {
		return 5, 0
	}

	probs := Softmax(logits[:5])
	best := 0
	for i := 1; i < 5; i++ {
		if probs[i] > probs[best] {
			best = i
		}
	}
	// id2label: 0→ESI-1, 1→ESI-2, 2→ESI-3, 3→ESI-4, 4→ESI-5
	return best + 1, float64(probs[best])
}

// Close releases the ONNX session resources.
func (p *Predictor) Close() {
	if p.session != nil {
		_ = p.session.Destroy()
	}
}
