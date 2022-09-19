#!/bin/bash
# Copyright 2022 Go Authors All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Start macOS VM.

if (( $# < 3 )); then
  echo "Usage: $0 <disk-image.qcow2> <OSK value> <VNC port index> [extra qemu args]"
  exit 1
fi

DISK=$1
OSK=$2
PORT=$3
EXTRA_ARGS=${@:4}

OVMF_CODE="$HOME/sysroot-macos-x86_64/share/qemu/edk2-x86_64-code.fd"

args=(
  -m 4096
  -cpu host
  -machine q35
  -usb -device usb-kbd -device usb-tablet
  # macOS only likes a power-of-two number of cores, but odd socket count is
  # fine.
  -smp cpus=6,sockets=3,cores=2,threads=1
  -device usb-ehci,id=ehci
  -device nec-usb-xhci,id=xhci
  -global nec-usb-xhci.msi=off
  -device isa-applesmc,osk="$OSK"
  -drive if=pflash,format=raw,readonly=on,file="$OVMF_CODE"
  -smbios type=2
  -device ich9-intel-hda -device hda-duplex
  -device ich9-ahci,id=sata
  -drive id=MacHDD,if=none,format=qcow2,file="$DISK"
  -device ide-hd,bus=sata.2,drive=MacHDD
  -netdev user,id=net0 # add ,hostfwd=tcp::5555-:22 to forward SSH to localhost:5555.
  -device virtio-net-pci,netdev=net0,id=net0,mac=52:54:00:c9:18:27
  -monitor stdio
  -device VGA,vgamem_mb=128
  -M accel=hvf
  -display vnc=127.0.0.1:"$PORT"
)

DYLD_LIBRARY_PATH="$HOME/sysroot-macos-x86_64/lib" "$HOME/sysroot-macos-x86_64/bin/qemu-system-x86_64" "${args[@]}" ${EXTRA_ARGS:+"$EXTRA_ARGS"}
