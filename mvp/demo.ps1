[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$projectDir = $PSScriptRoot
$demoDir = Join-Path ([System.IO.Path]::GetTempPath()) ("hearthmesh-demo-" + [guid]::NewGuid().ToString("N"))
$exe = Join-Path $demoDir "hearthmesh.exe"
$processes = @()
$locationPushed = $false

function Wait-ForCondition {
    param(
        [Parameter(Mandatory = $true)][scriptblock]$Condition,
        [Parameter(Mandatory = $true)][string]$FailureMessage,
        [int]$Attempts = 30
    )

    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        try {
            if (& $Condition) {
                return
            }
        }
        catch {
            # Transient startup and connection failures are expected while polling.
        }
        Start-Sleep -Seconds 1
    }
    throw $FailureMessage
}

function Start-HearthMeshNode {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$DataDir,
        [Parameter(Mandatory = $true)][string]$Listen,
        [Parameter(Mandatory = $true)][string]$Peers
    )

    $stdout = Join-Path $demoDir ("$Name.stdout.log")
    $stderr = Join-Path $demoDir ("$Name.stderr.log")
    $arguments = @(
        "serve",
        "-data", ('"{0}"' -f $DataDir),
        "-listen", $Listen,
        "-peers", $Peers
    )
    return Start-Process -FilePath $exe -ArgumentList $arguments -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
}

try {
    New-Item -ItemType Directory -Force -Path $demoDir | Out-Null
    $sourceDir = Join-Path $demoDir "source"
    $nodeADir = Join-Path $demoDir "node-a"
    $nodeBDir = Join-Path $demoDir "node-b"
    $nodeCDir = Join-Path $demoDir "node-c"
    New-Item -ItemType Directory -Force -Path $sourceDir, $nodeADir, $nodeBDir, $nodeCDir | Out-Null
    Set-Content -Path (Join-Path $sourceDir "hello.txt") -Value "A small, signed, public HearthPack." -Encoding UTF8

    Push-Location $projectDir
    $locationPushed = $true
    & go build -o $exe ./cmd/hearthmesh
    if ($LASTEXITCODE -ne 0) {
        throw "Go build failed."
    }

    $processes += Start-HearthMeshNode -Name "node-a" -DataDir $nodeADir -Listen "127.0.0.1:18081" -Peers "http://127.0.0.1:18082,http://127.0.0.1:18083"
    $processes += Start-HearthMeshNode -Name "node-b" -DataDir $nodeBDir -Listen "127.0.0.1:18082" -Peers "http://127.0.0.1:18081,http://127.0.0.1:18083"
    $processes += Start-HearthMeshNode -Name "node-c" -DataDir $nodeCDir -Listen "127.0.0.1:18083" -Peers "http://127.0.0.1:18081,http://127.0.0.1:18082"

    Wait-ForCondition -FailureMessage "Node A did not start." -Condition {
        $null = Invoke-RestMethod -Uri "http://127.0.0.1:18081/v1/status" -TimeoutSec 2
        return $true
    }

    $packPath = Join-Path $demoDir "pack.zip"
    $publisherKey = Join-Path $demoDir "publisher.key"
    & $exe pack -src $sourceDir -out $packPath -key $publisherKey
    if ($LASTEXITCODE -ne 0) {
        throw "Pack creation failed."
    }

    $verification = & $exe verify -pack $packPath
    if ($LASTEXITCODE -ne 0) {
        throw "Pack verification failed."
    }
    $packIdLine = $verification | Where-Object { $_ -like "verified pack ID:*" } | Select-Object -First 1
    if (-not $packIdLine) {
        throw "The verifier did not return a pack ID."
    }
    $packId = ($packIdLine -replace '^verified pack ID:\s*', '').Trim()

    & $exe publish -pack $packPath -node "http://127.0.0.1:18081"
    if ($LASTEXITCODE -ne 0) {
        throw "Publication to node A failed."
    }

    Wait-ForCondition -FailureMessage "Replication to nodes B and C timed out." -Condition {
        $packsB = @(Invoke-RestMethod -Uri "http://127.0.0.1:18082/v1/packs" -TimeoutSec 2)
        $packsC = @(Invoke-RestMethod -Uri "http://127.0.0.1:18083/v1/packs" -TimeoutSec 2)
        return ($packsB -contains $packId) -and ($packsC -contains $packId)
    }
    Write-Host "All three nodes verified the pack."

    Stop-Process -Id $processes[0].Id -Force
    Wait-Process -Id $processes[0].Id -ErrorAction SilentlyContinue
    Write-Host "Node A stopped. Nodes B and C retained the verified pack."

    $nodeBPack = Join-Path (Join-Path $nodeBDir "packs") ("$packId.zip")
    Set-Content -Path $nodeBPack -Value "deliberate corruption" -Encoding UTF8

    Wait-ForCondition -FailureMessage "Node B did not repair its corrupt pack." -Condition {
        & $exe verify -pack $nodeBPack *> $null
        return $LASTEXITCODE -eq 0
    }
    Write-Host "Node B repaired its corrupt copy from node C while node A remained offline."
    Write-Host "HearthMesh Windows demo passed."
}
finally {
    foreach ($process in $processes) {
        if ($null -ne $process -and -not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
    foreach ($process in $processes) {
        if ($null -ne $process) {
            Wait-Process -Id $process.Id -ErrorAction SilentlyContinue
        }
    }
    if ($locationPushed) {
        Pop-Location
    }
    if (Test-Path $demoDir) {
        Remove-Item -Path $demoDir -Recurse -Force
    }
}
