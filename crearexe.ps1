$wailsConfig = Get-Content "$PSScriptRoot\wails.json" -Raw | ConvertFrom-Json
$version = $wailsConfig.info.productVersion

$iscc = if (Test-Path "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe") {
    "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe"
} elseif (Test-Path "C:\Program Files (x86)\Inno Setup 6\ISCC.exe") {
    "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
} else {
    "ISCC.exe"
}

Write-Host "Generando instalador KatHub v$version..." -ForegroundColor Cyan
& $iscc "/DMyAppVersion=$version" "$PSScriptRoot\installer.iss"