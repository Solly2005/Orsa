package config

import "testing"

func TestNotebookModelDefaults(t *testing.T) {
	t.Setenv("BERT_BASE_MODEL_ID", "")
	t.Setenv("BERT_ESI_MODEL_DIR", "")
	t.Setenv("BERT_ESI_ONNX_DIR", "")
	t.Setenv("VISION_MODEL_ID", "")

	cfg := Load()
	if cfg.BertBaseModelID != "emilyalsentzer/Bio_ClinicalBERT" {
		t.Fatalf("unexpected BERT base model: %s", cfg.BertBaseModelID)
	}
	if cfg.BertModelDir != "./models/bert_esi" {
		t.Fatalf("unexpected BERT model dir: %s", cfg.BertModelDir)
	}
	if cfg.BertOnnxDir != "./models/bert_esi_onnx" {
		t.Fatalf("unexpected BERT ONNX dir: %s", cfg.BertOnnxDir)
	}
	if cfg.VisionModelID != "meta/Llama-3.2-90B-Vision-Instruct" {
		t.Fatalf("unexpected vision model: %s", cfg.VisionModelID)
	}
}

func TestOnnxRuntimeLibraryEnvAliases(t *testing.T) {
	t.Setenv("ONNX_RUNTIME_LIB", "")
	t.Setenv("ONNXRUNTIME_SHARED_LIBRARY_PATH", `C:\ort\onnxruntime.dll`)

	cfg := Load()
	if cfg.OnnxRuntimeLib != `C:\ort\onnxruntime.dll` {
		t.Fatalf("unexpected ONNX runtime lib from upstream alias: %s", cfg.OnnxRuntimeLib)
	}

	t.Setenv("ONNX_RUNTIME_LIB", `C:\custom\onnxruntime.dll`)

	cfg = Load()
	if cfg.OnnxRuntimeLib != `C:\custom\onnxruntime.dll` {
		t.Fatalf("ONNX_RUNTIME_LIB should override alias, got: %s", cfg.OnnxRuntimeLib)
	}
}
