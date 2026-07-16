# bounce-stress.ps1 — Windows stress test: 200+ simulated users browsing
# Usage: .\bounce-stress.ps1 [host] [users] [duration_secs]
#   host defaults to localhost:3001
#   users defaults to 200
#   duration defaults to 120 (seconds)

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

$global:total = 0
$global:ok = 0
$global:err = 0
$global:running = $true
$global:lock = [System.Threading.Mutex]::new()

function Browse-Worker {
    param($id)
    $rng = [Random]::new()
    while ($global:running) {
        foreach ($ep in $endpoints) {
            if (-not $global:running) { return }
            try {
                $url = "$BaseUrl$ep"
                $response = Invoke-WebRequest -Uri $url -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
                $global:lock.WaitOne() | Out-Null
                if ($response.StatusCode -eq 200) { $global:ok++ } else { $global:err++ }
                $global:total++
                $global:lock.ReleaseMutex()
            } catch {
                $global:lock.WaitOne() | Out-Null
                $global:err++
                $global:total++
                $global:lock.ReleaseMutex()
            }
            # Simulate read time (100-500ms)
            Start-Sleep -Milliseconds ($rng.Next(100, 500))
        }
    }
}

Write-Host "Launching $Users workers..." -ForegroundColor Yellow
$startTime = Get-Date
$deadline = $startTime.AddSeconds($Duration)

# Spawn workers in batches
$batchSize = 50
for ($i = 0; $i -lt $Users; $i += $batchSize) {
    $end = [Math]::Min($i + $batchSize, $Users)
    1..($end - $i) | ForEach-Object {
        Start-Job -ScriptBlock {
            param($id, $eps, $base)
            $global:running = $using:running
            $rng = [Random]::new()
            while ($using:running) {
                foreach ($ep in $using:eps) {
                    if (-not $using:running) { return }
                    try {
                        $r = Invoke-WebRequest -Uri "$using:base$ep" -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
                    } catch {}
                }
                Start-Sleep -Milliseconds ($rng.Next(100, 500))
            }
        } -ArgumentList $i, $endpoints, $BaseUrl
    }
    Start-Sleep -Milliseconds 200
}

# Display loop
$spinner = @('⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏')
while ((Get-Date) -lt $deadline) {
    $elapsed = [int]((Get-Date) - $startTime).TotalSeconds
    $remaining = $Duration - $elapsed
    $spin = $spinner[$elapsed % 10]
    Write-Host "`r$spin  ${elapsed}s | remaining ${remaining}s  " -NoNewline -ForegroundColor Cyan
    Start-Sleep -Milliseconds 500
}

$global:running = $false
Write-Host ""
Write-Host ""
Write-Host "═══ Done ═══" -ForegroundColor Green
$elapsed = [int]((Get-Date) - $startTime).TotalSeconds
Write-Host "  Duration: ${elapsed}s"
Write-Host "  Dashboard: $BaseUrl/dashboard" -ForegroundColor Yellow

# Cleanup jobs
Get-Job | Stop-Job
Get-Job | Remove-Job
