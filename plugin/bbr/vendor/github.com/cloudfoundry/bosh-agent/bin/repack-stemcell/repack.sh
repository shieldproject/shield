#!/bin/bash

set -e -x

env

stemcell_tgz=/tmp/stemcell.tgz
stemcell_dir=/tmp/stemcell
image_dir=/tmp/image

mkdir -p $stemcell_dir $image_dir
wget -O- $STEMCELL_URL > $stemcell_tgz
echo "$STEMCELL_SHA1  $stemcell_tgz" | shasum -c -

# Expose loopbacks in concourse container
(
  set -e
  mount_path=/tmp/self-cgroups
  cgroups_path=`cat /proc/self/cgroup|grep devices|cut -d: -f3`
  [ -d $mount_path ] && umount $mount_path && rmdir $mount_path
  mkdir -p $mount_path
  mount -t cgroup -o devices none $mount_path
  echo 'b 7:* rwm' > $mount_path/$cgroups_path/devices.allow
  umount $mount_path
  rmdir $mount_path
  for i in $(seq 0 260); do
  	mknod -m660 /dev/loop${i} b 7 $i 2>/dev/null || true
  done
)

# Repack stemcell
(
	set -e;
	cd $stemcell_dir
	tar xvf $stemcell_tgz
	new_ver=`date +%s`

	# Update stemcell with new agent
	(
		set -e;
		cd $image_dir
		tar xvf $stemcell_dir/image
		mnt_dir=/mnt/stemcell
		mkdir $mnt_dir
		mount -o loop,offset=32256 root.img $mnt_dir
		echo -n 0.0.${new_ver} > $mnt_dir/var/vcap/bosh/etc/stemcell_version
		cp /tmp/build/*/agent-src/bin/bosh-agent $mnt_dir/var/vcap/bosh/bin/bosh-agent

		if [ -n "$BOSH_DEBUG_PUB_KEY" ]; then
			sudo chroot $mnt_dir /bin/bash <<EOF
				useradd -m -s /bin/bash bosh_debug -G bosh_sudoers,bosh_sshers
				cd ~bosh_debug
				mkdir .ssh
				echo $BOSH_DEBUG_PUB_KEY >> .ssh/authorized_keys
				chmod go-rwx -R .
				chown -R bosh_debug:bosh_debug .
EOF
    fi

		umount $mnt_dir
		tar czvf $stemcell_dir/image *
	)

	sed -i.bak "s/version: .*/version: 0.0.${new_ver}/" stemcell.MF
	tar czvf $stemcell_tgz *
)

cp $stemcell_tgz /tmp/build/*/stemcell/
