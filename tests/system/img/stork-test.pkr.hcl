variables {
  ubuntu_ver = "20.04"
  ubuntu_code = "focal"
}

source "qemu" "stork-test" {
  accelerator         = "kvm"
  cpus                = "2"
  disk_discard        = "unmap"
  disk_image          = true
  disk_interface      = "virtio-scsi"
  disk_size           = "8G"
  format              = "qcow2"
  http_directory      = "cloud-data"
  iso_checksum        = "file:http://cloud-images.ubuntu.com/releases/${var.ubuntu_code}/release/SHA256SUMS"
  iso_url             = "http://cloud-images.ubuntu.com/releases/${var.ubuntu_code}/release/ubuntu-${var.ubuntu_ver}-server-cloudimg-amd64.img"
  memory              = "4096"
  qemuargs            = [["-smbios", "type=1,serial=ds=nocloud-net;instance-id=packer;seedfrom=http://{{ .HTTPIP }}:{{ .HTTPPort }}/"]]
  ssh_password        = "packerpassword"
  ssh_username        = "packer"
  use_default_display = true
}

build {
  sources = ["source.qemu.stork-test"]

  provisioner "file" {
    destination = "/tmp/lxd-init.yaml"
    source      = "lxd-init.yaml"
  }

  provisioner "file" {
    destination = "/tmp/network-config.yaml"
    source      = "network-config.yaml"
  }

  provisioner "shell" {
    inline        = [
      # enable tracing and exit on error
      "set -x -e",

      # install packages required for system tests
      "sudo /usr/bin/apt-get update",
      "sudo DEBIAN_FRONTEND=noninteractive /usr/bin/apt-get -y -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' --quiet=2 dist-upgrade",
      "sudo DEBIAN_FRONTEND=noninteractive /usr/bin/apt-get -y --no-install-recommends -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' --quiet=2 install apt-transport-https ca-certificates curl gnupg-agent software-properties-common python3-venv lxd rake net-tools firefox ruby ruby-dev rubygems build-essential git wget unzip",

      # install docker
      "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -",
      "sudo add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\"",
      "sudo apt-get update",
      "sudo DEBIAN_FRONTEND=noninteractive /usr/bin/apt-get -y --no-install-recommends -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' --quiet=2 install docker-ce docker-ce-cli containerd.io",

      # allow login using root account, this is used by gitlab libvirt custom runner which expects root with vagrant password
      "sudo bash -c 'echo -e \"vagrant\\nvagrant\" | passwd root'",
      "sudo sed -ie 's/.*PermitRootLogin.*/PermitRootLogin yes/g' /etc/ssh/sshd_config",

      # put correct netplan in place
      "sudo rm /etc/netplan/50-cloud-init.yaml",
      "sudo mv /tmp/network-config.yaml /etc/netplan/network-config.yaml",

      # generate hostkeys for ssh
      "sudo ssh-keygen -A",

      # initialize LXD
      "sudo lxd init --preseed < /tmp/lxd-init.yaml",

      # install gitlab-runner so it is possible to upload/download artifacts
      "wget 'https://gitlab-runner-downloads.s3.amazonaws.com/latest/deb/gitlab-runner_amd64.deb'",
      "sudo dpkg -i gitlab-runner_amd64.deb",

      # this is for testing
#      "git clone https://gitlab.isc.org/isc-projects/stork",
#      "python3 -m venv stork/tests/system/venv",
#      "stork/tests/system/venv/bin/pip install -U pip",
#      "stork/tests/system/venv/bin/pip install -r stork/tests/system/requirements.txt",
#      "cd stork && sudo rake system_tests"
    ]
    remote_folder = "/tmp"
  }

  provisioner "shell" {
    execute_command  = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    inline           = [
      # do lots of cleanup
      "/usr/bin/apt-get clean",
      #"rm -rf /etc/apparmor.d/cache/* /etc/apparmor.d/cache/.features /etc/netplan/50-cloud-init.yaml /etc/ssh/ssh_host* /etc/sudoers.d/90-cloud-init-users",
      "rm -rf /etc/apparmor.d/cache/* /etc/apparmor.d/cache/.features",
      "/usr/bin/truncate --size 0 /etc/machine-id",
      #"/usr/bin/gawk -i inplace '/PasswordAuthentication/ { gsub(/yes/, \"no\") }; { print }' /etc/ssh/sshd_config",
      "rm -rf /root/.ssh",
      "rm -f /snap/README",
      "find /usr/share/netplan -name __pycache__ -exec rm -r {} +",
      "rm -rf /var/cache/pollinate/seeded /var/cache/snapd/* /var/cache/motd-news",
      #"rm -rf /var/lib/cloud /var/lib/dbus/machine-id /var/lib/private /var/lib/systemd/timers /var/lib/systemd/timesync /var/lib/systemd/random-seed",
      "rm -f /var/lib/ubuntu-release-upgrader/release-upgrade-available",
      "rm -f /var/lib/update-notifier/fsck-at-reboot /var/lib/update-notifier/hwe-eol",
      "find /var/log -type f -exec rm {} +",
      "rm -rf /tmp/* /tmp/.*-unix /var/tmp/*",
      "rm -f /var/cache/apt/archives/*.deb /var/cache/apt/archives/partial/*.deb /var/cache/apt/*.bin || true",
      "userdel --force --remove packer",
      "/bin/sync",
      "/sbin/fstrim -v /"
    ]
    remote_folder    = "/tmp"
  }

}
