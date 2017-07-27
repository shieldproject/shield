Vagrant.configure('2') do |config|
  config.vm.box = 'cloudfoundry/bosh-lite'

  [:virtualbox, :vmware_fusion, :vmware_desktop, :vmware_workstation].each do |provider|
    config.vm.provider provider do |v, override|
      v.memory = 1024 * 4
      v.cpus = 4
    end
  end

  config.vm.provider :aws do |v, override|
    v.tags = { 'PipelineName' => 'bosh-init' }
    v.associate_public_ip = true

    v.access_key_id = ENV['BOSH_AWS_ACCESS_KEY_ID'] || ''
    v.secret_access_key = ENV['BOSH_AWS_SECRET_ACCESS_KEY'] || ''
    v.subnet_id = ENV['BOSH_LITE_SUBNET_ID'] || ''
    v.ami = ''
  end

  config.vm.synced_folder Dir.pwd, '/vagrant', disabled: true
  config.vm.provision :shell, inline: "mkdir -p /vagrant && chmod 777 /vagrant"
  config.vm.provision :shell, inline: "mkdir -p /var/vcap/sys/log/warden_cpi && chmod 777 /var/vcap/sys/log/warden_cpi"
end
