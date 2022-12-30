#!/bin/bash

set -eux

PKGS_DIR=$1

PKG_TYPES="deb rpm"

declare -A PKG_FILES
PKG_FILES=(
    ["isc-stork-agent-rpm"]="4"
    ["isc-stork-agent-deb"]="18"
    ["isc-stork-server-rpm"]="110"
    ["isc-stork-server-deb"]="128")

function cleanup {
    lxc stop "${cntr}" || true
    lxc delete "${cntr}" || true
}

trap cleanup ERR

for pkg_type in $PKG_TYPES; do
    if [ "${pkg_type}" = 'deb' ]; then
        cntr="u1804-stork"
        image="ubuntu/bionic/amd64"
        install="dpkg -i"
        get_pkg_files="dpkg -L"
    else
        cntr="c8-stork"
        image="centos/8/amd64"
        install="rpm -i"
        get_pkg_files="rpm -ql"
    fi

    cleanup

    lxc launch images:$image $cntr

    pkgs=$(ls "${PKGS_DIR}"/isc-stork*"${pkg_type}")
    for file in $pkgs; do
        lxc file push "${file}" "${cntr}/root/$(basename "${file}")"
    done
    lxc exec $cntr -- ls -al /root

    #lxc exec $cntr -- apt-get update

    for file in $pkgs; do
        lxc exec "${cntr}" -- "${install}" "/root/$(basename "${file}")"
        pkg_name=$(echo "${file}" | sed -n 's/.*\(isc-stork-[a-z]*\).*/\1/p')
        if [ "${pkg_name}" = "isc-stork-agent" ]; then
            lxc exec $cntr -- systemctl enable "${pkg_name}"
            lxc exec $cntr -- systemctl start "${pkg_name}"
            lxc exec $cntr -- systemctl status "${pkg_name}"
        fi

        # check number of files in installed package
        lines_cnt=$(lxc exec "${cntr}" -- "${get_pkg_files}" "${pkg_name}" | wc -l)
        echo "$pkg_name lines: $lines_cnt"
        if [ "${lines_cnt}" != "${PKG_FILES["${pkg_name}-${pkg_type}"]}" ]; then
            echo "wrong number of files in $pkg_type package $pkg_name"
            exit 1
        fi
    done

    cleanup
done
