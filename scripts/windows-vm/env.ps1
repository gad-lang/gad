# The PATH of the tools provision.ps1 installs, for the scripts that use them
# (dot-sourced: . C:\gad\env.ps1).
$env:Path = @(
    'C:\Program Files\Go\bin',
    "$env:USERPROFILE\go\bin",
    "$env:USERPROFILE\.bun\bin",
    'C:\ProgramData\mingw64\mingw64\bin',
    'C:\ProgramData\chocolatey\bin',
    'C:\Program Files\Git\cmd',
    'C:\Program Files\Git\usr\bin',
    'C:\Program Files\Git\bin',
    $env:Path
) -join ';'
