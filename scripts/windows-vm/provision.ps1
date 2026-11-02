# Installs in the VM what the CI's Windows job has (winvm.sh provision): Git
# for Windows (its bash runs make), make, gcc (mingw: go test -race), Go, Bun,
# staticcheck; git set as GitHub's Windows runners are (core.autocrlf), the
# submodules by https. Each tool already there, of its version, is skipped.
param(
    [Parameter(Mandatory)] [string] $Go,
    [Parameter(Mandatory)] [string] $Bun
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

function Step($what) { Write-Host "==> provision: $what" }

# what is tested runs here: no scan of each file it writes
try { Add-MpPreference -ExclusionPath 'C:\gad', "$env:USERPROFILE\go", "$env:LOCALAPPDATA\go-build", "$env:USERPROFILE\.bun" } catch { }

$choco = 'C:\ProgramData\chocolatey\bin\choco.exe'
if (-not (Test-Path $choco)) {
    Step 'Chocolatey'
    Invoke-Expression ((New-Object Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
}
foreach ($pkg in 'git', 'make', 'mingw') {
    if (-not (& $choco list --local-only --exact $pkg --limit-output)) {
        Step $pkg
        & $choco install -y --no-progress $pkg
        if ($LASTEXITCODE -ne 0) { throw "choco install $pkg: exit $LASTEXITCODE" }
    }
}

$goExe = 'C:\Program Files\Go\bin\go.exe'
if (-not (Test-Path $goExe) -or ((& $goExe version) -notmatch "go$([regex]::Escape($Go)) ")) {
    Step "Go $Go"
    $msi = "$env:TEMP\go$Go.msi"
    Invoke-WebRequest "https://go.dev/dl/go$Go.windows-amd64.msi" -OutFile $msi
    $p = Start-Process msiexec.exe -ArgumentList '/i', $msi, '/qn', '/norestart' -Wait -PassThru
    if ($p.ExitCode -ne 0) { throw "Go msi: exit $($p.ExitCode)" }
}

. C:\gad\env.ps1

$bunExe = "$env:USERPROFILE\.bun\bin\bun.exe"
if (-not (Test-Path $bunExe) -or ((& $bunExe --version) -ne $Bun)) {
    Step "Bun $Bun"
    & ([scriptblock]::Create((Invoke-RestMethod 'https://bun.sh/install.ps1'))) -Version $Bun
}

if (-not (Test-Path "$env:USERPROFILE\go\bin\staticcheck.exe")) {
    Step 'staticcheck'
    go install honnef.co/go/tools/cmd/staticcheck@latest
    if ($LASTEXITCODE -ne 0) { throw "staticcheck: exit $LASTEXITCODE" }
}

Step 'git (core.autocrlf, the submodules by https)'
git config --global core.autocrlf true
git config --global url.'https://github.com/'.insteadOf 'git@github.com:'
git config --global user.name 'gad winvm'
git config --global user.email 'winvm@localhost'

foreach ($t in @('git --version', 'make --version', 'gcc --version', 'go version', 'bun --version', 'staticcheck -version')) {
    $first = (Invoke-Expression $t 2>&1 | Select-Object -First 1)
    Write-Host "    $t → $first"
}
Step 'done'
