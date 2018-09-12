require 'json'
require 'open-uri'
require 'vagrant-aws'

home_dir="/home/ubuntu"
current_branch=`git rev-parse --abbrev-ref HEAD`
latest_head_commit=`git rev-parse HEAD`
latest_origin_commit=`git rev-parse origin/#{current_branch}`
clean_tree=system('git status | grep "nothing to commit"')
ssh_identities=`ssh-add -l`
stop_hours = 48  # if STOP_HOURS ENV is not specified, stop the instance after 2 days of running
if !ENV['STOP_HOURS'].nil?
  stop_hours = ENV['STOP_HOURS']
end

# Only run these checks on `vagrant up`
if ARGV[0] == "up"
  if ssh_identities.strip == 'The agent has no identities.'
    raise "No ssh identities are loaded, run `ssh-add` to load the private key that is allowed to clone the a2 repo!"
  end
  if !clean_tree
    puts %q(
      !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
      ! You have uncommitted changes that won't exist when we do the git clone on the remote EC2 instance !
      !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
    )
  end
  if latest_head_commit != latest_origin_commit
    puts %q(
      !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
      ! You have unpushed commits that won't exist when we do the git clone on the remote EC2 instance !
      !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
    )
  end

  if ENV['AWS_ACCESS_KEY_ID'].nil? || ENV['AWS_SECRET_ACCESS_KEY'].nil?
    raise "ENV variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be defined for this, aborting..."
  end

  if ENV['GITHUB_TOKEN'].nil?
    raise "ENV variable GITHUB_TOKEN must be defined for this, aborting..."
  end

  if ENV['AWS_SSH_KEY_NAME'].nil?
    raise "ENV variable AWS_SSH_KEY_NAME must be defined for this. See README.md for more details. Aborting..."
  end
end

def hab_version_from_manifest
  manifest = JSON.parse(open("https://packages.chef.io/manifests/dev/automate/latest.json").read)
  hab = manifest["hab"]
  hab.find {|x| x.start_with?("core/hab/") }.gsub("core/hab/", "")
end

$install_hab = <<SCRIPT
# Install latest habitat
curl --silent https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh | sudo bash -s -- -v #{hab_version_from_manifest}
# Install a specific version of habitat
# curl --silent https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh | sudo bash -s -- -v 0.54.0
SCRIPT

$install_victorias_bits = <<SCRIPT
apt-get install git -y
echo "* soft nofile 100000" >> /etc/security/limits.conf
echo "* hard nofile 256000" >> /etc/security/limits.conf
echo "root soft nofile 100000" >> /etc/security/limits.conf
echo "root hard nofile 256000" >> /etc/security/limits.conf
echo 'Defaults    env_keep += "SSH_AUTH_SOCK"' > /etc/sudoers.d/root_ssh_agent
SSHD_CONFIG="/etc/ssh/sshd_config"
if ! grep -q "^ClientAliveInterval" $SSHD_CONFIG; then
  echo "ClientAliveInterval 60" >> $SSHD_CONFIG
fi
if ! grep -q "^ClientAliveCountMax" $SSHD_CONFIG; then
  echo "ClientAliveCountMax 10000" >> $SSHD_CONFIG
fi
service ssh restart
CRON_FILE="/etc/cron.hourly/auto-stop"
if [ ! -f $CRON_FILE ]; then
cat<<'EOF' > $CRON_FILE
#!/bin/bash -e
uptime_hours=$(($(awk '{print int($1)}' /proc/uptime) / 3600))
# stop the instance if up for more than
if [ $uptime_hours -gt #{stop_hours} ] ; then
  wall "Automatically stopping instance after STOP_HOURS(#{stop_hours}) of uptime..."
  halt -p
fi
EOF
chmod +x $CRON_FILE
fi
SCRIPT

$github_clone_chef_load = <<SCRIPT
ssh-keyscan -H github.com >> ~/.ssh/known_hosts
cd #{home_dir}
git clone git@github.com:chef/chef-load.git
cd chef-load
echo "export GITHUB_TOKEN=\"#{ENV['GITHUB_TOKEN']}\"" > .secrets
git checkout #{latest_head_commit}

EC2HOSTNAME=`curl -Ss http://169.254.169.254/latest/meta-data/public-hostname`
#sed -i "s/fqdn = .*/fqdn = '$EC2HOSTNAME'/" dev/config.toml
SCRIPT

$enter_studio = <<SCRIPT
cat<<EOF >/etc/profile.d/hab_studio_setup.sh
  export GITHUB_TOKEN=#{ENV['GITHUB_TOKEN']}
  export HAB_STUDIO_SECRET_GITHUB_TOKEN=#{ENV['GITHUB_TOKEN']}
  export AWS_ACCESS_KEY_ID=#{ENV['AWS_ACCESS_KEY_ID']}
  export AWS_SECRET_ACCESS_KEY=#{ENV['AWS_SECRET_ACCESS_KEY']}
  export AZURE_CLIENT_ID=#{ENV['AZURE_CLIENT_ID']}
  export AZURE_CLIENT_SECRET=#{ENV['AZURE_CLIENT_SECRET']}
  export AZURE_TENANT_ID=#{ENV['AZURE_TENANT_ID']}

  cd #{home_dir}/chef-load
  source .envrc
  if [ ! -f ~/.hab/etc/cli.toml ]; then
    echo "Setting up HAB_ORIGIN=ubuntu"
    mkdir -p ~/.hab/etc
    cat<<'EOT' > ~/.hab/etc/cli.toml
origin = "ubuntu"
auth_token = "dev444BBB999"
ctl_secret = "dev333ZZZ111"
EOT
    hab origin key generate ubuntu
  fi
  hab studio run 'echo "http://$(curl -Ss http://169.254.169.254/latest/meta-data/public-hostname)" > url.txt'
  hab studio enter
EOF
STUDIORC="#{home_dir}/chef-load/.studiorc"
echo 'printf # Enable \'dep ensure\' in \'go_build\''
echo 'printf export RUN_GO_DEP=true'
echo 'printf "\033[0;31m>>> TWO MORE STEPS NEEDED TO RUN chef-load <<<\033[0m\n"' >> $STUDIORC
echo 'printf "1. Run this here:\033[1;32m hab pkg install chef/chef-load \033[0m\n"' >> $STUDIORC
echo 'printf "2. And then this:\033[1;34m hab pkg binlink chef/chef-load chef-load \033[0m\n"' >> $STUDIORC
SCRIPT

if ENV['AWS_SSH_KEY_PATH'].nil?
  ssh_key_path = '~/.ssh/id_rsa'
else
  ssh_key_path = ENV['AWS_SSH_KEY_PATH']
end

Vagrant.configure('2') do |config|
  config.vm.box = 'aws'
  config.vm.synced_folder ".", "/vagrant", disabled: true

  config.vm.provider 'aws' do |aws, override|
    aws.access_key_id = "#{ENV['AWS_ACCESS_KEY_ID']}"
    aws.secret_access_key = "#{ENV['AWS_SECRET_ACCESS_KEY']}"
    aws.keypair_name = ENV['AWS_SSH_KEY_NAME']
    aws.instance_type = 'm5.large'      # 2CPU, 8GB RAM
    # aws.instance_type = 'm5.xlarge'   # 4CPU, 16GB RAM
    # aws.instance_type = 'm5.2xlarge'  # 8CPU, 32GB RAM
    aws.region = 'us-east-2'            # US East (Ohio)
    aws.ami = 'ami-6a003c0f'            # Ubuntu 16.04 LTS in region 'us-east-2'
    aws.tags = {
      'Name' => "#{ENV['USER']}-chef_load-dev"
    }
    aws.security_groups = ['ssh-http-go-debug']
    override.ssh.username = 'ubuntu'
    override.ssh.private_key_path = ssh_key_path
  end

  config.ssh.forward_agent = true
  config.vm.provision 'shell', inline: $install_hab
  config.vm.provision 'shell', inline: $install_victorias_bits, :privileged => true
  config.vm.provision 'shell', inline: $github_clone_chef_load
  config.vm.provision 'shell', inline: $enter_studio
end
