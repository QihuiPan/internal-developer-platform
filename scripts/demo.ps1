$ErrorActionPreference = 'Stop'
$api = if ($env:PLATFORM_API_URL) { $env:PLATFORM_API_URL } else { 'http://localhost:8080' }
$headers = @{
  'X-Actor' = 'demo-user'
  'X-Role' = 'developer'
  'Idempotency-Key' = "payments-notifier-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
  'Content-Type' = 'application/json'
}
$descriptor = Get-Content -Raw -LiteralPath "$PSScriptRoot\..\examples\payments-notifier.json"
$created = Invoke-RestMethod -Method Post -Uri "$api/v1/services" -Headers $headers -Body $descriptor
$operationId = $created.operation.id
Write-Host "Accepted operation $operationId"

for ($attempt = 0; $attempt -lt 60; $attempt++) {
  $operation = Invoke-RestMethod -Method Get -Uri "$api/v1/operations/$operationId" -Headers @{'X-Actor'='demo-user'; 'X-Role'='developer'}
  Write-Host "$($operation.status): $($operation.steps.status -join ', ')"
  if ($operation.status -in @('SUCCEEDED', 'FAILED')) {
    $operation | ConvertTo-Json -Depth 8
    if ($operation.status -eq 'FAILED') { exit 1 }
    exit 0
  }
  Start-Sleep -Milliseconds 500
}

throw 'Operation did not reach a terminal state before the timeout.'
