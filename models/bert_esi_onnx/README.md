# BERT-ESI ONNX Runtime Artifacts

The notebook defines this directory as `ONNX_DIR` and the app should consume this folder.

Expected runtime files:

```text
onnx/model.onnx
tokenizer.json
config.json
tokenizer_config.json
special_tokens_map.json
vocab.txt
```

The label map in `config.json` must preserve:

```text
ESI-1, ESI-2, ESI-3, ESI-4, ESI-5
```

Large exported model files are intentionally not committed. Generate them by running the notebook export cells, then place/copy the output here.

If you have `models/bert_esi/model.safetensors` but not `onnx/model.onnx`, run:

```powershell
python scripts/export_bert_esi_onnx.py
```

after installing the dependencies listed in that script.
