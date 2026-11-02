# gad on Windows, in a local VM

`make test-windows` runs what the CI's Windows job runs — `make web-build test
build-cli generate-all` — on the working tree (what git tracks, its changes
included), in a VirtualBox VM of Windows Server 2022 (the evaluation ISO of
Microsoft, Server Core). The first time it does everything; after, each step
done is skipped:

| step | what | done when |
|---|---|---|
| `iso` | the Windows Server 2022 evaluation ISO (≈5 GB), its sha256 kept | `.winvm/state/iso.ok` |
| `additions` | VirtualBox's Guest Additions ISO, of the VirtualBox installed | `.winvm/state/additions.ok` |
| `vm` | the VM `gad-winvm`: 6 GB, 4 CPUs, a 64 GB disk (grows as used) | `.winvm/state/vm.ok` |
| `install` | Windows installed unattended (20–40 min), restarted once | `.winvm/state/install.ok` |
| `provision` | Git for Windows, make, mingw (gcc: `-race`), Go, Bun, staticcheck; a snapshot `provisioned` | the snapshot |
| `test` | the working tree checked out there as the CI does (`core.autocrlf`, its submodules by https), the targets run in Git's bash | every time |

Each test starts from the snapshot `provisioned`: the same machine every time.
Its log is kept in `.winvm/logs/test-*.log`.

```sh
make test-windows                                   # all of it
WINVM_TARGETS="test" make test-windows              # some targets only
scripts/windows-vm/winvm.sh status                  # what is done
scripts/windows-vm/winvm.sh start | stop            # the VM
WINVM_KEEP=1 scripts/windows-vm/winvm.sh test       # the VM left running after
scripts/windows-vm/winvm.sh destroy                 # the VM gone (the ISOs kept)
```

Everything is in `.winvm/` (git ignores it): the ISOs, the VM, its disk, the
logs and the password of the VM's user `gad` (`.winvm/state/password`, made
the first time). It needs VirtualBox (`VBoxManage`), `curl`, and the space:
≈5 GB of ISO, the disk as Windows and the tools fill it (≈20 GB).

| variable | default |
|---|---|
| `WINVM_DIR` | `.winvm` |
| `WINVM_NAME` | `gad-winvm` |
| `WINVM_MEMORY` / `WINVM_CPUS` / `WINVM_DISK_MB` | `6144` / `4` / `65536` |
| `WINVM_GO` / `WINVM_BUN` | the CI's: `1.26.9` / `1.3.14` |
| `WINVM_TARGETS` | `web-build test build-cli generate-all` |
| `WINVM_IMAGE` | `Windows Server 2022 Standard Evaluation` (Server Core) |
| `WINVM_ISO_SHA256` | the sha256 of the ISO Microsoft publishes: another refused |
| `WINVM_KEEP` | empty: the VM stopped after the test |

The evaluation of Windows Server runs 180 days; after, `winvm.sh destroy` and
`make test-windows` install it again (the ISO kept).
