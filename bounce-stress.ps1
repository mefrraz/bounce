# bounce-stress.ps1 — Aggressive stress test for Bounce
# Usage: .\bounce-stress.ps1 [host] [users] [duration_secs]
#   -Users 200   = number of parallel workers
#   -Duration 120 = test duration in seconds
# Examples:
#   .\bounce-stress.ps1                          # localhost:3001, 200 users, 120s
#   .\bounce-stress.ps1 192.168.1.200:3001 400 60  # 400 users, 60 seconds

param(
    [string]$HostAddr = "localhost:3001",
    [int]$Users = 200,
    [int]$Duration = 120
)

$BaseUrl = "http://$HostAddr"
$Aggressive = $Users -ge 300  # automatic aggressive mode for 300+ users

$endpoints = @(
    "/dashboard",
    "/docs", 
    "/test",
    "/health",
    "/metrics",
    "/api/competitions",
    "/api/games?club=119&season=2025/2026",
    "/api/games?club=127",
    "/api/games/today",
    "/api/games/live",
    "/api/standings/10902",
    "/api/elo",
    "/api/predictions/413420",
    "/api/h2h?team_a=127&team_b=120",
    "/api/athlete/269564",
    "/api/team/equipa_57682",
    "/api/club/127/teams",
    "/api/tugabasket/standings?competitionId=1",
    "/api/tugabasket/players?competitionId=1",
    "/api/tugabasket/teams?competitionId=1"
)

Write-Host "═══ Bounce Stress Test ═══" -ForegroundColor Green
Write-Host "Host: $BaseUrl   Users: $Users   Duration: ${Duration}s   Aggressive: $Aggressive" -ForegroundColor Cyan
Write-Host ""

# Aggressive: minimal delay, fast fire
if ($Aggressive) {
    $delayMin = 0
    $delayMax = 10
    $timeout = 3
} else {
    $delayMin = 50
    $delayMax = 200
    $timeout = 5
}

$scriptBlock = {
    param($base, $eps, $duration, $dMin, $dMax, $timeout)
    $deadline = (Get-Date).AddSeconds($duration)
    $rng = [Random]::new()
    $total = 0; $ok = 0; $err = 0
    while ((Get-Date) -lt $deadline) {
        foreach ($ep in $eps) {
            if ((Get-Date) -ge $deadline) { break }
            try {
                $r = Invoke-WebRequest -Uri "$base$ep" -TimeoutSec $timeout -UseBasicParsing -ErrorAction Stop
                if ($r.StatusCode -eq 200) { $ok++ } else { $err++ }
            } catch { $err++ }
            $total++
            if ($dMax -gt 0) {
                Start-Sleep -Milliseconds ($rng.Next($dMin, $dMax))
            }
        }
    }
    Write-Output "$total|$ok|$err"
}

Write-Host "Launching $Users workers (timeout=${timeout}s, delay=${delayMin}-${delayMax}ms)..." -ForegroundColor Yellow
$startTime = Get-Date

$jobs = @()
$batchSize = if ($Aggressive) { 50 } else { 25 }
for ($i = 0; $i -lt $Users; $i += $batchSize) {
    $end = [Math]::Min($i + $batchSize, $Users)
    for ($j = $i; $j -lt $end; $j++) {
        $job = Start-Job -ScriptBlock $scriptBlock -ArgumentList $BaseUrl, $endpoints, $Duration, $delayMin, $delayMax, $timeout
        $jobs += $job
    }
    Start-Sleep -Milliseconds 150
    $elapsed = [int]((Get-Date) - $startTime).TotalSeconds
    Write-Host "`r  Launched $end/$Users workers  (${elapsed}s)" -NoNewline
}
Write-Host ""

$spinner = @('⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏')
$lastTotal = 0
while ((Get-Date) -lt $startTime.AddSeconds($Duration)) {
    $elapsed = [int]((Get-Date) - $startTime).TotalSeconds
    $remaining = $Duration - $elapsed
    $running = ($jobs | Where-Object { $_.State -eq 'Running' }).Count
    $spin = $spinner[$elapsed % 10]
    # Quick RPS estimate (rough)
    $rpsEstimate = if ($Aggressive) { $running * 8 } else { $running * 2 }
    Write-Host "`r  $spin  ${elapsed}s | running=$running | ~$rpsEstimate req/s | remaining=${remaining}s  " -NoNewline -ForegroundColor Cyan
    Start-Sleep -Milliseconds 500
}
Start-Sleep -Seconds 2  # let stragglers finish
Write-Host ""

# Collect results
$totalAll = 0; $okAll = 0; $errAll = 0
foreach ($job in $jobs) {
    $result = Receive-Job -Job $job -ErrorAction SilentlyContinue
    if ($result) {
        $parts = $result -split '\|'
        if ($parts.Count -ge 3) {
            $totalAll += [int]$parts[0]
            $okAll += [int]$parts[1]
            $errAll += [int]$parts[2]
        }
    }
    Remove-Job -Job $job -Force
}

$elapsed = [int]((Get-Date) - $startTime).TotalSeconds
Write-Host ""
Write-Host "═══ Done ═══" -ForegroundColor Green
Write-Host "  Duration:       ${elapsed}s"
Write-Host "  Total requests: $totalAll" -ForegroundColor Cyan
Write-Host "  OK:             $okAll" -ForegroundColor Green
Write-Host "  Errors:         $errAll" -ForegroundColor Red
if ($elapsed -gt 0 -and $totalAll -gt 0) {
    $rps = [Math]::Round($totalAll / $elapsed)
    Write-Host "  Avg req/s:      $rps" -ForegroundColor Yellow
}
Write-Host ""
Write-Host "Dashboard: $BaseUrl/dashboard" -ForegroundColor Yellow
