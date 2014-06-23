Vagrant.configure("2") do |config|
  ## Choose your base box
  config.vm.box = "precise64"
  config.vm.box_url = "http://files.vagrantup.com/precise64.box"
  config.ssh.forward_agent = true

  config.vm.network "forwarded_port", guest: 8000, host: 8000

  ## For masterless, mount your file roots file root
  config.vm.synced_folder "conf/salt/roots/", "/srv/"
  config.vm.synced_folder ".", "/usr/local/julython"

  ## Set your salt configs here
  config.vm.provision :salt do |salt|

    ## Minion config is set to ``file_client: local`` for masterless
    salt.minion_config = "conf/salt/minion"

    ## Installs our example formula in "conf/salt/roots/salt"
    salt.run_highstate = true

  end

  VAGRANT_POOL_NAME = ENV['VAGRANT_LIBVIRT_POOL'] || "default"
  config.vm.provider :libvirt do |libvirt, override|
    libvirt.storage_pool_name = VAGRANT_POOL_NAME
  end
end
