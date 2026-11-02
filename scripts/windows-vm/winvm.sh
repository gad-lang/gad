#!/usr/bin/env bash
# Tests gad on Windows in a local VirtualBox VM — the steps of the CI's
# Windows job (.github/workflows/workflow.yml) —, from the download of Windows
# to the last test, each step it already did skipped the next time:
#
#   iso        the Windows Server 2022 evaluation ISO (Microsoft), its hash
#   additions  VirtualBox's Guest Additions ISO, of its version
#   vm         the VM (WINVM_NAME), its disk in WINVM_DIR
#   install    Windows installed, unattended — Server Core, the Guest Additions
#   provision  Git, make, gcc (mingw), Go, Bun, staticcheck; a snapshot of it
#   test       the working tree (what git tracks, changes included) checked out
#              there as the CI does (core.autocrlf), its submodules, and
#              make web-build test build-cli generate-all (WINVM_TARGETS)
#
#   winvm.sh [all|iso|additions|vm|install|provision|test|start|stop|status|destroy]
#
# make test-windows runs `all`. Each test starts from the snapshot
# "provisioned": the same machine every time. See README.md.
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$here" rev-parse --show-toplevel)

: "${WINVM_DIR:=$repo/.winvm}"
: "${WINVM_NAME:=gad-winvm}"
: "${WINVM_MEMORY:=6144}"
: "${WINVM_CPUS:=4}"
: "${WINVM_DISK_MB:=65536}"
: "${WINVM_USER:=gad}"
: "${WINVM_GO:=1.26.9}"
: "${WINVM_BUN:=1.3.14}"
: "${WINVM_TARGETS:=web-build test build-cli generate-all}"
: "${WINVM_KEEP:=}" # not empty: the VM left running after the test
: "${WINVM_ISO_URL:=https://go.microsoft.com/fwlink/p/?LinkID=2195280&clcid=0x409&culture=en-us&country=US}"
: "${WINVM_ISO_SHA256:=3e4fa6d8507b554856fc9ca6079cc402df11a8b79344871669f0251535255325}"
# the edition installed: Server Core (no desktop), the lightest
: "${WINVM_IMAGE:=Windows Server 2022 Standard Evaluation}"
: "${WINVM_INSTALL_TIMEOUT:=5400}" # seconds the installation may take

state=$WINVM_DIR/state
iso=$WINVM_DIR/SERVER_EVAL_x64FRE_en-us.iso
vbox_version=$(VBoxManage --version | sed -E 's/^([0-9]+\.[0-9]+\.[0-9]+).*/\1/')
additions=$WINVM_DIR/VBoxGuestAdditions_$vbox_version.iso
disk=$WINVM_DIR/$WINVM_NAME.vdi
mkdir -p "$state" "$WINVM_DIR/logs"

log() { printf '==> %s\n' "$*" >&2; }
done_mark() { touch "$state/$1.ok"; }
is_done() { [ -f "$state/$1.ok" ]; }

password() {
	if [ ! -f "$state/password" ]; then
		(umask 077 && printf 'Gad-%s!' "$(head -c 12 /dev/urandom | od -An -tx1 | tr -d ' \n')" > "$state/password")
	fi
	cat "$state/password"
}

vm_exists() { VBoxManage showvminfo "$WINVM_NAME" >/dev/null 2>&1; }
vm_state() { VBoxManage showvminfo "$WINVM_NAME" --machinereadable 2>/dev/null | sed -n 's/^VMState="\(.*\)"/\1/p'; }
has_snapshot() { VBoxManage snapshot "$WINVM_NAME" list --machinereadable 2>/dev/null | grep -q "=\"$1\"$"; }

# guest runs a program in the VM as WINVM_USER: guest EXE ARG… (its output
# here, its exit code the command's).
guest() {
	local exe=$1
	shift
	VBoxManage guestcontrol "$WINVM_NAME" run --username "$WINVM_USER" --password "$(password)" \
		--wait-stdout --wait-stderr --exe "$exe" -- "$(basename "${exe//\\//}")" "$@"
}
guest_ps() { guest 'C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe' -NoProfile -ExecutionPolicy Bypass "$@"; }
copy_to() { VBoxManage guestcontrol "$WINVM_NAME" copyto --username "$WINVM_USER" --password "$(password)" "$1" "$2"; }
copy_from() { VBoxManage guestcontrol "$WINVM_NAME" copyfrom --username "$WINVM_USER" --password "$(password)" "$1" "$2"; }

start_vm() {
	check_host
	case "$(vm_state)" in
	running) ;;
	*) log "starting $WINVM_NAME"; VBoxManage startvm "$WINVM_NAME" --type headless >/dev/null ;;
	esac
}

stop_vm() {
	[ "$(vm_state)" = running ] || return 0
	log "stopping $WINVM_NAME"
	VBoxManage controlvm "$WINVM_NAME" acpipowerbutton >/dev/null 2>&1 || true
	for _ in $(seq 1 60); do
		[ "$(vm_state)" = running ] || return 0
		sleep 5
	done
	VBoxManage controlvm "$WINVM_NAME" poweroff >/dev/null 2>&1 || true
	sleep 3
}

# wait_guest waits for a program to run in the VM (the Guest Additions up,
# the user's logon possible), at most $1 seconds.
wait_guest() {
	local limit=${1:-900} start=$SECONDS
	log "waiting for Windows (at most ${limit}s)"
	until guest 'C:\Windows\System32\cmd.exe' /c exit 0 >/dev/null 2>&1; do
		if [ $((SECONDS - start)) -ge "$limit" ]; then
			log "Windows did not answer in ${limit}s"
			return 1
		fi
		sleep 20
	done
	log "Windows answers (after $((SECONDS - start))s)"
}

# check_host refuses a host whose CPU has its virtualization off — no svm
# (AMD-V) nor vmx (Intel VT-x) in /proc/cpuinfo —: VirtualBox cannot start a
# VM there (VERR_SVM_DISABLED, VERR_VMX_DISABLED). It is turned on in the
# BIOS/UEFI: "SVM Mode" (AMD) or "Intel Virtualization Technology".
check_host() {
	command -v VBoxManage >/dev/null || { log "no VBoxManage: install VirtualBox"; exit 1; }
	if [ -r /proc/cpuinfo ] && ! grep -qwE 'svm|vmx' /proc/cpuinfo; then
		log "the CPU's virtualization is off (no svm/vmx in /proc/cpuinfo): turn it on in the BIOS/UEFI —"
		log "  \"SVM Mode\" (AMD) or \"Intel Virtualization Technology\" (Intel) —, then run this again"
		exit 1
	fi
}

step_iso() {
	if is_done iso && [ -f "$iso" ]; then log "iso: done"; return; fi
	log "iso: downloading the Windows Server 2022 evaluation (≈5 GB) to $iso"
	curl -fL --retry 5 -C - -o "$iso.part" "$WINVM_ISO_URL"
	mv "$iso.part" "$iso"
	local sum
	sum=$(sha256sum "$iso" | cut -d' ' -f1)
	if [ -n "$WINVM_ISO_SHA256" ] && [ "$sum" != "$WINVM_ISO_SHA256" ]; then
		log "iso: sha256 $sum, want $WINVM_ISO_SHA256"
		rm -f "$iso"
		exit 1
	fi
	log "iso: sha256 $sum"
	echo "$sum" > "$state/iso.sha256"
	done_mark iso
}

step_additions() {
	if is_done additions && [ -f "$additions" ]; then log "additions: done"; return; fi
	log "additions: VBoxGuestAdditions $vbox_version"
	curl -fL --retry 5 -o "$additions.part" \
		"https://download.virtualbox.org/virtualbox/$vbox_version/VBoxGuestAdditions_$vbox_version.iso"
	mv "$additions.part" "$additions"
	done_mark additions
}

step_vm() {
	check_host
	if is_done vm && vm_exists; then log "vm: done"; return; fi
	if vm_exists; then
		log "vm: $WINVM_NAME exists, not made by this script's last run: destroy it first (winvm.sh destroy)"
		exit 1
	fi
	log "vm: $WINVM_NAME — ${WINVM_MEMORY} MB, $WINVM_CPUS CPUs, a ${WINVM_DISK_MB} MB disk"
	VBoxManage createvm --name "$WINVM_NAME" --ostype Windows2022_64 --basefolder "$WINVM_DIR" --register >/dev/null
	VBoxManage modifyvm "$WINVM_NAME" --memory "$WINVM_MEMORY" --cpus "$WINVM_CPUS" --vram 64 \
		--graphicscontroller vboxsvga --nic1 nat --audio-enabled off --usb-ohci off \
		--clipboard-mode disabled --ioapic on --rtc-use-utc on
	VBoxManage createmedium disk --filename "$disk" --size "$WINVM_DISK_MB" --format VDI >/dev/null
	VBoxManage storagectl "$WINVM_NAME" --name SATA --add sata --controller IntelAhci --portcount 4
	VBoxManage storageattach "$WINVM_NAME" --storagectl SATA --port 0 --device 0 --type hdd --medium "$disk"
	done_mark vm
}

# image_index is the index of the image WINVM_IMAGE in the ISO — its lines
# ImageIndexN="NAME (VERSION / ARCH / LANG)" —; empty when it has none.
image_index() {
	VBoxManage unattended detect --iso="$iso" --machine-readable 2>/dev/null |
		sed -nE 's/^ImageIndex([0-9]+)="(.*) \([^()]*\)"$/\1\t\2/p' |
		awk -F'\t' -v want="$WINVM_IMAGE" '$2 == want { print $1; exit }'
}

step_install() {
	if is_done install; then log "install: done"; return; fi
	check_host
	if [ "$(vm_state)" != running ]; then
		local index
		index=$(image_index)
		if [ -z "$index" ]; then
			log "install: no image \"$WINVM_IMAGE\" in the ISO:"
			VBoxManage unattended detect --iso="$iso" | sed -n 's/^ *Image #/  #/p' >&2 || true
			exit 1
		fi
		log "install: Windows (image $index, \"$WINVM_IMAGE\"), unattended — 20 to 40 minutes"
		# the user an administrator with no UAC filter: what guestcontrol runs
		# (installers, Defender's exclusions) needs the whole token
		VBoxManage unattended install "$WINVM_NAME" --iso="$iso" --image-index="$index" \
			--user="$WINVM_USER" --password="$(password)" --full-user-name="$WINVM_USER" \
			--install-additions --additions-iso="$additions" \
			--time-zone=UTC --hostname="$WINVM_NAME.local" --locale=en_US --country=US \
			--post-install-command='cmd /c reg add HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System /v EnableLUA /t REG_DWORD /d 0 /f' \
			--start-vm=headless >"$WINVM_DIR/logs/unattended.log"
	fi
	wait_guest "$WINVM_INSTALL_TIMEOUT"
	# the setting of UAC holds once Windows starts again
	log "install: restarting Windows once"
	guest 'C:\Windows\System32\shutdown.exe' /r /t 5 >/dev/null 2>&1 || true
	sleep 60
	wait_guest 900
	eject_media
	done_mark install
}

# eject_media takes the installation's ISOs out of the VM's drives.
eject_media() {
	VBoxManage showvminfo "$WINVM_NAME" --machinereadable |
		sed -nE 's/^"([^"-]+)-([0-9]+)-([0-9]+)"=".*\.(iso|viso)"$/\1 \2 \3/p' |
		while read -r ctl port dev; do
			VBoxManage storageattach "$WINVM_NAME" --storagectl "$ctl" --port "$port" --device "$dev" \
				--medium emptydrive >/dev/null 2>&1 || true
		done
}

step_provision() {
	if is_done provision && has_snapshot provisioned; then log "provision: done"; return; fi
	start_vm
	wait_guest 900
	log "provision: Git, make, mingw, Go $WINVM_GO, Bun $WINVM_BUN, staticcheck"
	guest_ps -Command "New-Item -ItemType Directory -Force C:\\gad | Out-Null"
	copy_to "$here/env.ps1" 'C:\gad\env.ps1'
	copy_to "$here/provision.ps1" 'C:\gad\provision.ps1'
	guest_ps -File 'C:\gad\provision.ps1' -Go "$WINVM_GO" -Bun "$WINVM_BUN"
	stop_vm
	log "provision: snapshot \"provisioned\""
	if has_snapshot provisioned; then VBoxManage snapshot "$WINVM_NAME" delete provisioned >/dev/null; fi
	VBoxManage snapshot "$WINVM_NAME" take provisioned --description "gad: Windows with the CI's tools (winvm.sh)" >/dev/null
	done_mark provision
}

# bundle is the working tree — what git tracks, its changes included (git
# stash create) — as a git bundle of the branch winvm-test.
bundle() {
	local commit out=$WINVM_DIR/gad.bundle
	commit=$(git -C "$repo" stash create 2>/dev/null || true)
	commit=${commit:-$(git -C "$repo" rev-parse HEAD)}
	git -C "$repo" update-ref refs/heads/winvm-test "$commit"
	git -C "$repo" bundle create "$out" refs/heads/winvm-test >/dev/null 2>&1
	git -C "$repo" update-ref -d refs/heads/winvm-test
	if [ -n "$(git -C "$repo" ls-files --others --exclude-standard)" ]; then
		log "test: files git does not track are not sent (git add them to test them)"
	fi
	echo "$out"
}

step_test() {
	if has_snapshot provisioned; then
		stop_vm
		log "test: from the snapshot \"provisioned\""
		VBoxManage snapshot "$WINVM_NAME" restore provisioned >/dev/null
	fi
	start_vm
	wait_guest 900
	local b stamp code=0
	b=$(bundle)
	stamp=$(date +%Y%m%d-%H%M%S)
	log "test: $(git -C "$repo" rev-parse --short HEAD) (and the changes), make $WINVM_TARGETS"
	copy_to "$b" 'C:\gad\gad.bundle'
	copy_to "$here/env.ps1" 'C:\gad\env.ps1'
	copy_to "$here/run-tests.ps1" 'C:\gad\run-tests.ps1'
	copy_to "$here/run-tests.sh" 'C:\gad\run-tests.sh'
	guest_ps -File 'C:\gad\run-tests.ps1' -Targets "$WINVM_TARGETS" || code=$?
	copy_from 'C:\gad\test.log' "$WINVM_DIR/logs/test-$stamp.log" >/dev/null 2>&1 || true
	log "test: the log in $WINVM_DIR/logs/test-$stamp.log"
	if [ -z "$WINVM_KEEP" ]; then stop_vm; fi
	if [ "$code" -ne 0 ]; then
		log "test: FAILED (exit $code)"
		return "$code"
	fi
	log "test: OK"
}

status() {
	for s in iso additions vm install provision; do
		if is_done $s; then echo "$s: done"; else echo "$s: to do"; fi
	done
	vm_exists && echo "vm state: $(vm_state)" || echo "vm state: none"
	has_snapshot provisioned && echo "snapshot: provisioned" || true
}

destroy() {
	if vm_exists; then
		stop_vm
		VBoxManage unregistervm "$WINVM_NAME" --delete >/dev/null
	fi
	rm -f "$state"/vm.ok "$state"/install.ok "$state"/provision.ok "$disk"
	log "destroyed $WINVM_NAME (the ISOs kept in $WINVM_DIR)"
}

case "${1:-all}" in
all) step_iso; step_additions; step_vm; step_install; step_provision; step_test ;;
iso) step_iso ;;
additions) step_additions ;;
vm) step_vm ;;
install) step_install ;;
provision) step_provision ;;
test) step_test ;;
start) start_vm; wait_guest 900 ;;
stop) stop_vm ;;
status) status ;;
destroy) destroy ;;
*) sed -n '2,23p' "$0" >&2; exit 2 ;;
esac
