Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/bionic64"

  config.vm.provider "virtualbox" do |v|
    v.memory = 4096
  end

  config.vm.provision "shell", inline: "apt-get install -y rake openjdk-11-jre-headless"
end
