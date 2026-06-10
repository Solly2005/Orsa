# Immutable Notebook Workflow Contract

The notebook is the authority for chatbot and triage behavior. Production code must preserve:

- `LOOP_CAP = 5`
- `MAX_MESSAGES = 20`
- State keys: chief complaint, symptoms, onset, severity, location, modifiers, demographics, vitals, risk factors, red flags, attachment summaries, turn count, and messages.
- Flow: M0 attachments, M1 scope, M2 extraction, state merge, UMLS coding, red-flag fast path, M3 sufficiency, M4 clarify, BERT prediction, M5 triage, escalate-only reconciliation, M6 patient response, FHIR bundle.
- Reconciliation: final ESI is the most urgent level among BERT, GPT-OSS, and red-flag/vitals floor. Silent de-escalation is forbidden.
- Attachment behavior: unreadable text, blurry images, and uncertainty must be surfaced without adding new medical decision-making.
- BERT-ESI model defaults are taken from the notebook: base model `emilyalsentzer/Bio_ClinicalBERT`, fine-tuned checkpoint `models/bert_esi`, ONNX runtime artifacts `models/bert_esi_onnx`, labels `ESI-1` through `ESI-5`.
- GPT-OSS uses Hugging Face Router with model `openai/gpt-oss-120b`.
- Vision uses GitHub Models/Marketplace with model `meta/Llama-3.2-90B-Vision-Instruct`.

Persona/profile extraction must remain store-only until a separately approved workflow explicitly changes this contract.
