# bounce-stress.ps1 — Windows stress test: 200+ simulated users browsing
# Usage: .\bounce-stress.ps1 [host] [users] [duration_secs]
# Requires PowerShell 5.1+

param(
    [string]$HostAddr = "localhost:3001",
    [int]$Users = 200,
    [int]$Duration = 120
)

$BaseUrl = "http://$HostAddr"

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
Write-Host "Host: $BaseUrl   Users: $Users   Duration: ${Duration}s" -ForegroundColor Cyan
Write-Host ""

$scriptBlock = {
    param($base, $eps, $duration)
    $deadline = (Get-Date).AddSeconds($duration)
    $rng = [Random]::new()
    $total = 0; $ok = 0
    while ((Get-Date) -lt $deadline) {
        foreach ($ep in $eps) {
            if ((Get-Date) -ge $deadline) { break }
            try {
                $r = Invoke-WebRequest -Uri "$base$ep" -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
                if ($r.StatusCode -eq 200) { $ok++ }
            } catch {}
            $total++
            Start-Sleep -Milliseconds ($rng.Next(100, 500))
        }
    }
    Write-Output "$total|$ok"
}

Write-Host "Launching $Users workers..." -ForegroundColor Yellow
$startTime = Get-Date

$jobs = @()
$batchSize = 25
for ($i = 0; $i -lt $Users; $i += $batchSize) {
    $end = [Math]::Min($i + $batchSize, $Users)
    for ($j = $i; $j -lt $end; $j++) {
        $job = Start-Job -ScriptBlock $scriptBlock -ArgumentList $BaseUrl, $endpoints, $Duration
        $jobs += $job
    }
    Start-Sleep -Milliseconds 200
    $elapsed = [int]((Get-Date) - $startTime).TotalSeconds
    Write-Host "`rLaunched $end/$Users workers  (${elapsed}s)" -NoNewline
}
Write-Host ""

$spinner = @('⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏')
while ((Get-Date) -lt $startTime.AddSeconds($Duration)) {
    $elapsed = [int]((Get-Date) - $startTime).TotalSeconds
    $remaining = $Duration - $elapsed
    $running = ($jobs | Where-Object { $_.State -eq 'Running' }).Count
    $spin = $spinner[$elapsed % 10]
    Write-Host "`r$spin  ${elapsed}s | running=$running | remaining=${remaining}s  " -NoNewline -ForegroundColor Cyan
    Start-Sleep -Milliseconds 500
}
Write-Host ""

# Collect results
$totalAll = 0; $okAll = 0
foreach ($job in $jobs) {
    $result = Receive-Job -Job $job -ErrorAction SilentlyContinue
    if ($result) {
        $parts = $result -split '\|'
        if ($parts.Count -eq 2) {
            $totalAll += [int]$parts[0]
            $okAll += [int]$parts[1]
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
Write-Host "  Errors:         $($totalAll - $okAll)" -ForegroundColor Red
if ($elapsed -gt 0) {
    Write-Host "  Avg req/s:      $([Math]::Round($totalAll / $elapsed))"
}
Write-Host ""
Write-Host "Dashboard: $BaseUrl/dashboard" -ForegroundColor Yellow
