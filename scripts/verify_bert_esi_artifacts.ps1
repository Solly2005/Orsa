$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$required = @(
  "models\bert_esi\model.safetensors",
  "models\bert_esi\config.json",
  "models\bert_esi\tokenizer.json",
  "models\bert_esi\tokenizer_config.json",
  "models\bert_esi\special_tokens_map.json",
  "models\bert_esi\vocab.txt",
  "models\bert_esi_onnx\config.json",
  "models\bert_esi_onnx\tokenizer.json",
  "models\bert_esi_onnx\tokenizer_config.json",
  "models\bert_esi_onnx\special_tokens_map.json",
  "models\bert_esi_onnx\vocab.txt",
  "models\bert_esi_onnx\onnx\model.onnx"
)

$missing = foreach ($item in $required) {
  $path = Join-Path $root $item
  if (!(Test-Path -LiteralPath $path)) { $item }
}

if ($missing.Count -gt 0) {
  Write-Host "Missing BERT-ESI artifacts:"
  $missing | ForEach-Object { Write-Host " - $_" }
  exit 1
}

Write-Host "All BERT-ESI artifacts are present."
