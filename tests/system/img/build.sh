#!/usr/bin/env bash
set -e -x

PACKER_VER=1.7.2
UBUNTU_VER=20.04
UBUNTU_CODE=focal

# cleanup
rm -rf output* stork-tests-*.qcow2

# get packer if missing
if [ ! -x packer ]; then
    rm -f packer*zip packer
    wget https://releases.hashicorp.com/packer/${PACKER_VER}/packer_${PACKER_VER}_linux_amd64.zip
    unzip packer_${PACKER_VER}_linux_amd64.zip
    rm packer_${PACKER_VER}_linux_amd64.zip
fi

# build image
./packer build -var "ubuntu_ver=${UBUNTU_VER}" -var "ubuntu_code=${UBUNTU_CODE}" stork-test.pkr.hcl

# compress image
TMPDIR=~/tmp virt-sparsify --compress output-stork-test/packer-stork-test stork-tests-ubuntu-${UBUNTU_VER}-x86_64.qcow2

# wrap in docker image
docker build -f docker.txt --build-arg UBUNTU_VER=${UBUNTU_VER} -t registry.gitlab.isc.org/isc-projects/images/qcow2:stork-tests-ubuntu-${UBUNTU_VER}-x86_64 .

# upload to gitlab
docker push registry.gitlab.isc.org/isc-projects/images/qcow2:stork-tests-ubuntu-${UBUNTU_VER}-x86_64
