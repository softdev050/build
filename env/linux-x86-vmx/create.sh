#!/bin/sh
# Copyright 2019 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# This creates the debian-bullseye-vmx buildlet VM that's
# like the Container-Optimized OS but using Debian Bullseye
# instead of the Chromium OS, and with nested virtualization
# enabled.

set -e
set -x

ZONE=us-central1-f
TARGET_IMAGE=debian-bullseye-vmx

TMP_DISK=dev-debian-vmx-tmpdisk
TMP_IMG=dev-debian-vmx-image
TMP_VM=dev-debian-vmx

# Create disk, forking Debian 9 (Stretch).
gcloud compute disks delete $TMP_DISK --zone=$ZONE --quiet || true
gcloud compute disks create $TMP_DISK \
       --zone=$ZONE \
       --size=40GB \
       --image-project=debian-cloud \
       --image-family debian-11

# Create image based on that disk, with the nested virtualization
# opt-in flag ("license").
gcloud compute images delete $TMP_IMG --quiet || true
gcloud compute images create \
       $TMP_IMG \
       --source-disk=$TMP_DISK \
       --source-disk-zone=$ZONE \
       --licenses "https://www.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx"

# No longer need that temp disk:
gcloud compute disks delete $TMP_DISK --zone=$ZONE --quiet

# Create the VM
gcloud compute instances delete --zone=$ZONE $TMP_VM --quiet || true
gcloud compute instances create \
       $TMP_VM \
       --zone=$ZONE \
       --image=$TMP_IMG \
       --min-cpu-platform "Intel Haswell" \
       --network default-vpc \
       --no-service-account --no-scopes

echo "Waiting for SSH port to be available..."
while ! gcloud compute ssh $TMP_VM --zone=$ZONE --tunnel-through-iap -- echo hi; do
    sleep 1
done

echo "SSH is up. Copying prep-vm.sh script to VM..."

# gcloud compute scp lacks an --internal-ip flag, even though gcloud
# compute ssh has it. Annoying. Workaround:
gcloud compute scp --zone=$ZONE --tunnel-through-iap prep-vm.sh $TMP_VM:

# And prep the machine.
gcloud compute ssh $TMP_VM --zone=$ZONE --tunnel-through-iap -- sudo bash ./prep-vm.sh

echo "Done prepping machine; shutting down"

# Shut it down so it's a stable source to snapshot from.
gcloud compute instances stop $TMP_VM --zone=$ZONE

# Now make the new image from our instance's disk.
gcloud compute images delete $TARGET_IMAGE --quiet || true
gcloud compute images create $TARGET_IMAGE --source-disk=$TMP_VM --source-disk-zone=$ZONE

gcloud compute images delete $TMP_IMG --quiet

echo "Done."
