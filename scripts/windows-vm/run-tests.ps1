# The test in the VM (winvm.sh test): the bundle of the working tree checked
# out as the CI's Windows job does — CRLF by core.autocrlf, its submodules —
# and each make target of -Targets run in Git's bash, as there; the output in
# C:\gad\test.log, the exit code the first target's that failed.
param(
    [string] $Targets = 'web-build test build-cli generate-all'
)
$ErrorActionPreference = 'Stop'
. C:\gad\env.ps1

$src = 'C:\gad\src'
if (Test-Path $src) { Remove-Item -Recurse -Force $src }
git clone -q -b winvm-test C:\gad\gad.bundle $src
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Set-Location $src
git submodule update --init --recursive -q
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$targetList = $Targets -split '\s+' | Where-Object { $_ }
& 'C:\Program Files\Git\bin\bash.exe' -lc "/c/gad/run-tests.sh $($targetList -join ' ') 2>&1 | tee /c/gad/test.log; exit `${PIPESTATUS[0]}"
exit $LASTEXITCODE
