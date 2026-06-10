# BERT-ESI Fine-Tuned Weights

The notebook defines this directory as `MODEL_DIR`.

Expected source model:

```text
emilyalsentzer/Bio_ClinicalBERT
```

The notebook fine-tunes that base model for five ESI labels and saves the best checkpoint here with:

```python
model.save_pretrained(MODEL_DIR)
tokenizer.save_pretrained(MODEL_DIR)
```

This directory should contain the fine-tuned Hugging Face checkpoint files after the notebook training/export run. Large model binaries are intentionally not committed.

Current expected files:

```text
model.safetensors
config.json
tokenizer.json
tokenizer_config.json
special_tokens_map.json
vocab.txt
```
