Vagrant.configure("2") do |config|
  ## Choose your base box
  config.vm.box = "precise64"
  config.vm.box_url = "http://files.vagrantup.com/precise64_vmware.box"

  ## For masterless, mount your file roots file root
  config.vm.synced_folder "conf/salt/roots/", "/srv/"
  config.vm.synced_folder ".", "/usr/local/julython"

  # setup other paths
  
  ## Set your salt configs here
  config.vm.provision :salt do |salt|

    ## Minion config is set to ``file_client: local`` for masterless
    salt.minion_config = "conf/salt/minion"

    ## Installs our example formula in "conf/salt/roots/salt"
    salt.run_highstate = true

  end
end
