chcp 65001 > $null
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$BASE = "http://localhost:8000/api/v1"
$tmpFile = "$env:TEMP\checkin_test_body.json"

function Api {
    param([string]$Method, [string]$Path, [hashtable]$Body, [string]$Token)
    $uri = "$BASE$Path"
    $curlArgs = @("--silent", "--max-time", "10", "-X", $Method, $uri, "-H", "Content-Type: application/json")
    if ($Token) { $curlArgs += @("-H", "Authorization: Bearer $Token") }
    if ($Body) {
        $Body | ConvertTo-Json -Compress | Out-File -FilePath $tmpFile -Encoding utf8
        $curlArgs += @("-d", "@$tmpFile")
    }
    $result = & curl.exe @curlArgs 2>&1
    return $result
}

$ts = (Get-Date).ToString("HHmmss")
$user = "fix_test_$ts"

Write-Host "`n===== 1. Register =====" -ForegroundColor Cyan
$reg = Api -Method POST -Path "/users" -Body @{ username=$user; password="Abc123456"; email="$user@test.com"; comfirmPassword="Abc123456" }
Write-Host $reg

Write-Host "`n===== 2. Login =====" -ForegroundColor Cyan
$login = Api -Method POST -Path "/auth/login" -Body @{ username=$user; password="Abc123456" }
Write-Host $login
$token = ($login | ConvertFrom-Json).data.accessToken
if (-not $token) { Write-Host "Login failed!" -ForegroundColor Red; exit 1 }
Write-Host "Token OK"

Write-Host "`n===== 3. Daily check-in =====" -ForegroundColor Cyan
Write-Host (Api -Method POST -Path "/checkins" -Token $token)

Write-Host "`n===== 4. Check points (should be 1) =====" -ForegroundColor Cyan
Write-Host (Api -Method GET -Path "/points/summary" -Token $token)

$ym = (Get-Date).ToString("yyyy-MM")
Write-Host "`n===== 5. Calendar ($ym) - Bug1 fix verify =====" -ForegroundColor Cyan
Write-Host (Api -Method GET -Path "/checkins/calendar?yearMonth=$ym" -Token $token)

$yesterday = (Get-Date).AddDays(-1).ToString("yyyy-MM-dd")
$today     = (Get-Date).ToString("yyyy-MM-dd")
$tomorrow  = (Get-Date).AddDays(1).ToString("yyyy-MM-dd")

Write-Host "`n===== 6. Retro yesterday ($yesterday) - should FAIL (only 1 point, need 10) =====" -ForegroundColor Yellow
Write-Host (Api -Method POST -Path "/checkins/retroactive" -Body @{ date=$yesterday } -Token $token)

Write-Host "`n===== 7. Retro today ($today) - should FAIL (cannot retro today) =====" -ForegroundColor Yellow
Write-Host (Api -Method POST -Path "/checkins/retroactive" -Body @{ date=$today } -Token $token)

Write-Host "`n===== 8. Retro tomorrow ($tomorrow) - should FAIL (future) =====" -ForegroundColor Yellow
Write-Host (Api -Method POST -Path "/checkins/retroactive" -Body @{ date=$tomorrow } -Token $token)

Write-Host "`n===== 9. Retro other month - should FAIL =====" -ForegroundColor Yellow
Write-Host (Api -Method POST -Path "/checkins/retroactive" -Body @{ date="2026-03-15" } -Token $token)

Write-Host "`n===== Done! =====" -ForegroundColor Green
Remove-Item $tmpFile -ErrorAction SilentlyContinue
