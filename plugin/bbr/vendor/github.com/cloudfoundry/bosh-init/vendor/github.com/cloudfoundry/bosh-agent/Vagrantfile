Vagrant.configure('2') do |config|
  config.vm.box = 'cloudfoundry/bosh-lite'
  config.vm.box_version = '9000.20.0'

  config.vm.provider :virtualbox do |v, override|
    # To use a different IP address for the bosh-lite director, uncomment this line:
    # override.vm.network :private_network, ip: '192.168.59.4', id: :local
  end

  config.vm.provider :aws do |v, override|
    v.associate_public_ip = true
    # To turn off public IP echoing, uncomment this line:
    # override.vm.provision :shell, id: "public_ip", run: "always", inline: "/bin/true"

    # To turn off CF port forwarding, uncomment this line:
    # override.vm.provision :shell, id: "port_forwarding", run: "always", inline: "/bin/true"
    v.tags = {
      'PipelineName' => 'bosh-agent'
    }
  end

  agent_dir = '/home/vagrant/go/src/github.com/cloudfoundry/bosh-agent'

  config.vm.synced_folder '.', agent_dir, type: "rsync"

#  config.vm.synced_folder Dir.pwd, '/vagrant', disabled: true
  config.vm.provision :shell, inline: "mkdir -p /vagrant && chmod 777 /vagrant"
  config.vm.provision :shell, inline: "chmod 777 /var/vcap/sys/log/cpi"

  config.vm.provision :shell, inline: "sudo #{agent_dir}/integration/assets/install-go.sh"
  config.vm.provision :shell, inline: "sudo #{agent_dir}/integration/assets/install-agent.sh"
  config.vm.provision :shell, inline: "sudo #{agent_dir}/integration/assets/install-fake-registry.sh"
  config.vm.provision :shell, inline: "sudo #{agent_dir}/integration/assets/disable_growpart.sh"
end
