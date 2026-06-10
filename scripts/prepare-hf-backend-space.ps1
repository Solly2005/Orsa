param(
  [string]$OutputPath = "dist\huggingface-backend-space"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$distRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot "dist"))
$outputFull = [IO.Path]::GetFullPath((Join-Path $repoRoot $OutputPath))

if (!$outputFull.StartsWith($distRoot, [StringComparison]::OrdinalIgnoreCase)) {
  throw "OutputPath must stay inside the repo dist directory. Got: $outputFull"
}

if (Test-Path -LiteralPath $outputFull) {
  Remove-Item -LiteralPath $outputFull -Recurse -Force
}

New-Item -ItemType Directory -Force -Path $outputFull | Out-Null

function Copy-RequiredItem {
  param(
    [Parameter(Mandatory = $true)][string]$Source,
    [Parameter(Mandatory = $true)][string]$Destination
  )

  $sourceFull = Join-Path $repoRoot $Source
  if (!(Test-Path -LiteralPath $sourceFull)) {
    throw "Required deployment source is missing: $Source"
  }

  $destinationFull = Join-Path $outputFull $Destination
  $destinationParent = Split-Path -Parent $destinationFull
  if ($destinationParent) {
    New-Item -ItemType Directory -Force -Path $destinationParent | Out-Null
  }

  Copy-Item -LiteralPath $sourceFull -Destination $destinationFull -Recurse -Force
}

Copy-RequiredItem "services" "services"
Copy-RequiredItem "proto" "proto"
Copy-RequiredItem "models\bert_esi_onnx" "models\bert_esi_onnx"
Copy-RequiredItem "deploy\huggingface\backend" "deploy\huggingface\backend"
Copy-RequiredItem "deploy\huggingface\backend\Dockerfile" "Dockerfile"
Copy-RequiredItem "deploy\huggingface\backend\README.md" "README.md"
Copy-RequiredItem "deploy\huggingface\backend\.gitattributes" ".gitattributes"
Copy-RequiredItem "deploy\huggingface\backend\.dockerignore" ".dockerignore"

$onnxRoot = Join-Path $outputFull "models\bert_esi_onnx\model.onnx"
$onnxNested = Join-Path $outputFull "models\bert_esi_onnx\onnx\model.onnx"
if (!(Test-Path -LiteralPath $onnxRoot) -and !(Test-Path -LiteralPath $onnxNested)) {
  Write-Warning "No BERT ONNX model file found in the generated Space. The Go service will still start, but BERT-ESI will fall back to the safe neutral signal."
}

Write-Host "Hugging Face backend Space prepared at:"
Write-Host "  $outputFull"
Write-Host ""
Write-Host "Next:"
Write-Host "  1. Create a Docker Space on Hugging Face."
Write-Host "  2. Push the contents of this directory to that Space repository."
Write-Host "  3. Set the required Space secrets and variables."
