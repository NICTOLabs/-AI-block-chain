param(
    [int]$RelayPort = 8788,
    [string]$RelayDir = "C:\Users\BYU\Desktop\TENDER\AI-block-chain\go-chain\relay-worker",
    [string]$NodeExe = "C:\Users\BYU\Desktop\TENDER\AI-block-chain\go-chain\tender-node.exe",
    [string]$DataBase = "C:\Users\BYU\AppData\Local\Temp\opencode\relay-e2e"
)

$ErrorActionPreference = "Stop"

# Kill any leftover tender-node/relay processes
Get-Process tender-node, node -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$relay = Start-Process node -ArgumentList "local-relay.js" -WorkingDirectory $RelayDir -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 3
Write-Output "relay pid=$($relay.Id)"
$relayURL = "http://localhost:$RelayPort"

$node1Data = "$DataBase\node1"
$node2Data = "$DataBase\node2"
Remove-Item -Recurse -Force $node1Data, $node2Data -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $node1Data, $node2Data | Out-Null

$node1 = Start-Process $NodeExe -ArgumentList @("-api-port","9101","-p2p-port","9301","-data-dir",$node1Data,"-consensus","pos","-api-key","key1","-node-id","node1","-relay-url",$relayURL) -PassThru -WindowStyle Hidden
$node2 = Start-Process $NodeExe -ArgumentList @("-api-port","9102","-p2p-port","9302","-data-dir",$node2Data,"-consensus","pos","-api-key","key2","-node-id","node2","-relay-url",$relayURL) -PassThru -WindowStyle Hidden

Write-Output "node1 pid=$($node1.Id) node2 pid=$($node2.Id)"
Start-Sleep -Seconds 6

function Invoke-Health($port) {
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:$port/health" -UseBasicParsing -TimeoutSec 5
        return "HTTP $($r.StatusCode)"
    } catch { return "DOWN: $($_.Exception.Message)" }
}

Write-Output "node1 health: $(Invoke-Health 9101)"
Write-Output "node2 health: $(Invoke-Health 9102)"

function Post-Json($url, $body, $key) {
    (Invoke-WebRequest -Uri $url -Method Post -Body ($body | ConvertTo-Json -Depth 10) -ContentType "application/json" -Headers @{ Authorization = "Bearer $key" } -UseBasicParsing -TimeoutSec 10).Content | ConvertFrom-Json
}
function Get-Json($url) {
    (Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 5).Content | ConvertFrom-Json
}
function Try-Get($url) {
    try { return Get-Json $url } catch { return $null }
}
function ChainHeight($port) {
    @((Get-Json "http://localhost:$port/api/chain").chain).Count
}

Write-Output "initial heights: n1=$(ChainHeight 9101) n2=$(ChainHeight 9102)"

# 1. Create wallet on node1
$wallet = Post-Json "http://localhost:9101/api/wallet" @{} "key1"
$walletAddr = $wallet.address
Write-Output "wallet created on node1: $walletAddr"

# 2. Faucet funds alice on node1 (broadcasts state over relay)
Post-Json "http://localhost:9101/api/faucet" @{ address = $walletAddr; amount = 200000000 } "key1" | Out-Null
$acct2 = $null
for ($i = 0; $i -lt 10; $i++) {
    Start-Sleep -Seconds 2
    $acct2 = Try-Get "http://localhost:9102/api/account?address=$walletAddr"
    if ($acct2) { break }
}
if (-not $acct2) { Write-Output "TIMEOUT: node2 never saw alice account" }
Write-Output "node2 sees alice balance after state-sync: $($acct2.balance)"

# 3. Mine on node1 (broadcasts block over relay)
$mined = Post-Json "http://localhost:9101/api/mine" @{} "key1"
Write-Output "node1 mine result: $($mined | ConvertTo-Json -Compress -Depth 5)"
Start-Sleep -Seconds 6

Write-Output "heights after mine: n1=$(ChainHeight 9101) n2=$(ChainHeight 9102)"

# Diagnostics: peer counts and relay state
Write-Output "--- monitoring ---"
Write-Output "node1: $((Get-Json 'http://localhost:9101/api/monitoring') | ConvertTo-Json -Compress)"
Write-Output "node2: $((Get-Json 'http://localhost:9102/api/monitoring') | ConvertTo-Json -Compress)"
Write-Output "relay peers: $((Invoke-WebRequest -Uri 'http://localhost:8788/peers' -UseBasicParsing).Content)"

$h1 = ChainHeight 9101
$h2 = ChainHeight 9102
Write-Output "node1 alice balance: $((Get-Json "http://localhost:9101/api/account?address=$walletAddr").balance)"
Write-Output "node2 alice balance: $((Get-Json "http://localhost:9102/api/account?address=$walletAddr").balance)"

Stop-Process -Id $node1.Id, $node2.Id, $relay.Id -Force -ErrorAction SilentlyContinue

if ($h1 -ge 2 -and $h2 -ge 2 -and $acct2.balance -eq 200000000) {
    Write-Output "E2E RELAY TEST PASSED (both nodes at height >= 2, node2 synced state)"
} else {
    Write-Output "E2E RELAY TEST FAILED (heights n1=$h1 n2=$h2, synced=$($acct2.balance))"
}
