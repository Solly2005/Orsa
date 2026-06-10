"""Export the notebook's fine-tuned BERT-ESI checkpoint to ONNX.

Run from the repository root after installing:

    pip install torch transformers optimum[onnxruntime] onnx onnxruntime safetensors
    python scripts/export_bert_esi_onnx.py
"""

from __future__ import annotations

import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MODEL_DIR = ROOT / "models" / "bert_esi"
ONNX_DIR = ROOT / "models" / "bert_esi_onnx"


def main() -> None:
    try:
        from optimum.exporters.onnx import main_export
    except ImportError as exc:
        raise SystemExit(
            "Missing export dependencies. Install with: "
            "pip install torch transformers optimum[onnxruntime] onnx onnxruntime safetensors"
        ) from exc

    required = [
        MODEL_DIR / "model.safetensors",
        MODEL_DIR / "config.json",
        MODEL_DIR / "tokenizer.json",
        MODEL_DIR / "tokenizer_config.json",
        MODEL_DIR / "special_tokens_map.json",
        MODEL_DIR / "vocab.txt",
    ]
    missing = [str(path) for path in required if not path.exists()]
    if missing:
        raise SystemExit("Missing BERT-ESI checkpoint files:\n" + "\n".join(missing))

    ONNX_DIR.mkdir(parents=True, exist_ok=True)
    main_export(
        model_name_or_path=str(MODEL_DIR),
        output=str(ONNX_DIR),
        task="text-classification",
        opset=18,
    )

    onnx_file = ONNX_DIR / "model.onnx"
    nested_onnx_file = ONNX_DIR / "onnx" / "model.onnx"
    if onnx_file.exists() and not nested_onnx_file.exists():
        nested_onnx_file.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(onnx_file), str(nested_onnx_file))

    for name in ["config.json", "tokenizer.json", "tokenizer_config.json", "special_tokens_map.json", "vocab.txt"]:
        source = MODEL_DIR / name
        target = ONNX_DIR / name
        if source.exists():
            shutil.copy2(source, target)

    if not nested_onnx_file.exists():
        raise SystemExit(f"ONNX export did not produce expected file: {nested_onnx_file}")

    print(f"Exported BERT-ESI ONNX model to {nested_onnx_file}")


if __name__ == "__main__":
    main()
