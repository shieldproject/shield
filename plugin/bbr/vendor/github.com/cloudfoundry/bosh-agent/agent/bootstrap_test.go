package agent_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	fakeip "github.com/cloudfoundry/bosh-agent/platform/net/ip/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	fakedisk "github.com/cloudfoundry/bosh-agent/platform/disk/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	sigar "github.com/cloudfoundry/gosigar"

	devicepathresolver "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshcdrom "github.com/cloudfoundry/bosh-agent/platform/cdrom"
	boshcert "github.com/cloudfoundry/bosh-agent/platform/cert"
	boshdisk "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshnet "github.com/cloudfoundry/bosh-agent/platform/net"
	bosharp "github.com/cloudfoundry/bosh-agent/platform/net/arp"
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshudev "github.com/cloudfoundry/bosh-agent/platform/udevdevice"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshsigar "github.com/cloudfoundry/bosh-agent/sigar"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

func init() {
	Describe("bootstrap", func() {
		Describe("Run", func() {
			var (
				platform    *fakeplatform.FakePlatform
				dirProvider boshdir.Provider

				settingsService *fakesettings.FakeSettingsService
			)

			BeforeEach(func() {
				platform = fakeplatform.NewFakePlatform()
				dirProvider = boshdir.NewProvider("/var/vcap")
				settingsService = &fakesettings.FakeSettingsService{}
			})

			bootstrap := func() error {
				logger := boshlog.NewLogger(boshlog.LevelNone)
				return NewBootstrap(platform, dirProvider, settingsService, logger).Run()
			}

			It("sets up runtime configuration", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupRuntimeConfigurationWasInvoked).To(BeTrue())
			})

			Describe("SSH tunnel setup for registry", func() {
				It("returns error without configuring ssh on the platform if getting public key fails", func() {
					settingsService.PublicKeyErr = errors.New("fake-get-public-key-err")

					err := bootstrap()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-get-public-key-err"))

					Expect(platform.SetupSSHCalled).To(BeFalse())
				})

				Context("when public key is not empty", func() {
					BeforeEach(func() {
						settingsService.PublicKey = "fake-public-key"
					})

					It("gets the public key and sets up ssh via the platform", func() {
						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())

						Expect(platform.SetupSSHPublicKey).To(ConsistOf("fake-public-key"))
						Expect(platform.SetupSSHUsername).To(Equal("vcap"))
					})

					It("returns error if configuring ssh on the platform fails", func() {
						platform.SetupSSHErr = errors.New("fake-setup-ssh-err")

						err := bootstrap()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-setup-ssh-err"))
					})
				})

				Context("when public key key is empty", func() {
					BeforeEach(func() {
						settingsService.PublicKey = ""
					})

					It("gets the public key and does not setup SSH", func() {
						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())

						Expect(platform.SetupSSHCalled).To(BeFalse())
					})
				})

				Context("when the environment has authorized keys", func() {
					BeforeEach(func() {
						settingsService.Settings.Env.Bosh.AuthorizedKeys = []string{"fake-public-key", "another-fake-public-key"}
						settingsService.PublicKey = ""
					})

					It("gets the public key and sets up SSH", func() {
						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())

						Expect(platform.SetupSSHCalled).To(BeTrue())
						Expect(platform.SetupSSHPublicKey).To(ConsistOf("fake-public-key", "another-fake-public-key"))
						Expect(platform.SetupSSHUsername).To(Equal("vcap"))
					})
				})

				Context("when both have authorized keys", func() {
					BeforeEach(func() {
						settingsService.Settings.Env.Bosh.AuthorizedKeys = []string{"another-fake-public-key"}
						settingsService.PublicKey = "fake-public-key"
					})

					It("gets the public key and sets up SSH", func() {
						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())

						Expect(platform.SetupSSHCalled).To(BeTrue())
						Expect(platform.SetupSSHPublicKey).To(ConsistOf("fake-public-key", "another-fake-public-key"))
						Expect(platform.SetupSSHUsername).To(Equal("vcap"))
					})
				})
			})

			It("sets up ipv6", func() {
				settingsService.Settings.Env.Bosh.IPv6.Enable = true

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupIPv6Config).To(Equal(boshsettings.IPv6{Enable: true}))
			})

			It("sets up hostname", func() {
				settingsService.Settings.AgentID = "foo-bar-baz-123"

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupHostnameHostname).To(Equal("foo-bar-baz-123"))
			})

			It("fetches initial settings", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(settingsService.SettingsWereLoaded).To(BeTrue())
			})

			It("returns error from loading initial settings", func() {
				settingsService.LoadSettingsError = errors.New("fake-load-error")

				err := bootstrap()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-load-error"))
			})

			Context("load settings errors", func() {
				BeforeEach(func() {
					settingsService.LoadSettingsError = errors.New("fake-load-error")
					settingsService.PublicKey = "fake-public-key"
				})

				It("sets a ssh key despite settings error", func() {
					err := bootstrap()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-load-error"))
					Expect(platform.SetupSSHCalled).To(BeTrue())
				})
			})

			It("sets up networking", func() {
				networks := boshsettings.Networks{
					"bosh": boshsettings.Network{},
				}
				settingsService.Settings.Networks = networks

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupNetworkingNetworks).To(Equal(networks))
			})

			It("sets up ephemeral disk", func() {
				var swapSize uint64
				swapSize = 2048
				settingsService.Settings.Env.Bosh.SwapSizeInMB = &swapSize
				settingsService.Settings.Disks = boshsettings.Disks{
					Ephemeral: "fake-ephemeral-disk-setting",
				}

				platform.GetEphemeralDiskPathRealPath = "/dev/sda"

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupEphemeralDiskWithPathDevicePath).To(Equal("/dev/sda"))
				Expect(*platform.SetupEphemeralDiskWithPathSwapSize).To(Equal(uint64(2048 * 1024 * 1024)))
				Expect(platform.GetEphemeralDiskPathSettings).To(Equal(boshsettings.DiskSettings{
					VolumeID: "fake-ephemeral-disk-setting",
					Path:     "fake-ephemeral-disk-setting",
				}))
			})

			It("returns error if setting ephemeral disk fails", func() {
				platform.SetupEphemeralDiskWithPathErr = errors.New("fake-setup-ephemeral-disk-err")
				err := bootstrap()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-setup-ephemeral-disk-err"))
			})

			It("sets up raw ephemeral disks if paths exist", func() {
				settingsService.Settings.Disks = boshsettings.Disks{
					RawEphemeral: []boshsettings.DiskSettings{{Path: "/dev/xvdb"}, {Path: "/dev/xvdc"}},
				}

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupRawEphemeralDisksCallCount).To(Equal(1))
				Expect(len(platform.SetupRawEphemeralDisksDevices)).To(Equal(2))
				Expect(platform.SetupRawEphemeralDisksDevices[0].Path).To(Equal("/dev/xvdb"))
				Expect(platform.SetupRawEphemeralDisksDevices[1].Path).To(Equal("/dev/xvdc"))
			})

			It("returns error if setting raw ephemeral disks fails", func() {
				platform.SetupRawEphemeralDisksErr = errors.New("fake-setup-raw-ephemeral-disks-err")
				err := bootstrap()
				Expect(platform.SetupRawEphemeralDisksCallCount).To(Equal(1))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-setup-raw-ephemeral-disks-err"))
			})

			It("sets up data dir", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupDataDirCalled).To(BeTrue())
			})

			It("returns error if set up of data dir fails", func() {
				platform.SetupDataDirErr = errors.New("fake-setup-data-dir-err")
				err := bootstrap()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-setup-data-dir-err"))
			})

			It("sets up tmp dir", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupTmpDirCalled).To(BeTrue())
			})

			It("sets up /home dir", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupHomeDirCalled).To(BeTrue())
			})

			It("sets up log dir", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupLogDirCalled).To(BeTrue())
			})

			It("sets up logging and auditing services", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupLoggingAndAuditingCalled).To(BeTrue())
			})

			It("returns error if set up of tmp dir fails", func() {
				platform.SetupTmpDirErr = errors.New("fake-setup-tmp-dir-err")
				err := bootstrap()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-setup-tmp-dir-err"))
			})

			It("grows the root filesystem", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupRootDiskCalledTimes).To(Equal(1))
			})

			It("returns an error if growing the root filesystem fails", func() {
				platform.SetupRootDiskError = errors.New("growfs failed")

				err := bootstrap()
				Expect(err).To(HaveOccurred())
				Expect(platform.SetupRootDiskCalledTimes).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring("growfs failed"))
			})

			It("sets root and vcap passwords", func() {
				settingsService.Settings.Env.Bosh.Password = "some-encrypted-password"
				settingsService.Settings.Env.Bosh.KeepRootPassword = false

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(2).To(Equal(len(platform.UserPasswords)))
				Expect("some-encrypted-password").To(Equal(platform.UserPasswords["root"]))
				Expect("some-encrypted-password").To(Equal(platform.UserPasswords["vcap"]))
			})

			It("does not change root password if keep_root_password is set to true", func() {
				settingsService.Settings.Env.Bosh.Password = "some-encrypted-password"
				settingsService.Settings.Env.Bosh.KeepRootPassword = true

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(1).To(Equal(len(platform.UserPasswords)))
				Expect("some-encrypted-password").ToNot(Equal(platform.UserPasswords["root"]))
				Expect("some-encrypted-password").To(Equal(platform.UserPasswords["vcap"]))
			})

			It("sets ntp", func() {
				settingsService.Settings.Ntp = []string{
					"0.north-america.pool.ntp.org",
					"1.north-america.pool.ntp.org",
				}

				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(2).To(Equal(len(platform.SetTimeWithNtpServersServers)))
				Expect("0.north-america.pool.ntp.org").To(Equal(platform.SetTimeWithNtpServersServers[0]))
				Expect("1.north-america.pool.ntp.org").To(Equal(platform.SetTimeWithNtpServersServers[1]))
			})

			It("setups up monit user", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.SetupMonitUserSetup).To(BeTrue())
			})

			It("starts monit", func() {
				err := bootstrap()
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.StartMonitStarted).To(BeTrue())
			})

			Describe("RemoveDevTools", func() {

				It("removes development tools if settings.env.bosh.remove_dev_tools is true", func() {
					settingsService.Settings.Env.Bosh.RemoveDevTools = true
					platform.GetFs().WriteFileString(path.Join(dirProvider.EtcDir(), "dev_tools_file_list"), "/usr/bin/gfortran")

					err := bootstrap()
					Expect(err).NotTo(HaveOccurred())
					Expect(platform.IsRemoveDevToolsCalled).To(BeTrue())
					Expect(platform.PackageFileListPath).To(Equal(path.Join(dirProvider.EtcDir(), "dev_tools_file_list")))
				})

				It("does NOTHING if settings.env.bosh.remove_dev_tools is NOT set", func() {
					err := bootstrap()
					Expect(err).NotTo(HaveOccurred())
					Expect(platform.IsRemoveDevToolsCalled).To(BeFalse())
				})

				It("does NOTHING if if settings.env.bosh.remove_dev_tools is true AND dev_tools_file_list does NOT exist", func() {
					settingsService.Settings.Env.Bosh.RemoveDevTools = true
					err := bootstrap()
					Expect(err).NotTo(HaveOccurred())
					Expect(platform.IsRemoveDevToolsCalled).To(BeFalse())
				})
			})

			Describe("RemoveStaticLibraries", func() {
				It("removes development tools if settings.env.bosh.remove_static_libraries is true", func() {
					settingsService.Settings.Env.Bosh.RemoveStaticLibraries = true
					platform.GetFs().WriteFileString(path.Join(dirProvider.EtcDir(), "static_libraries_list"), "/usr/lib/libsupp.a")

					err := bootstrap()
					Expect(err).NotTo(HaveOccurred())
					Expect(platform.IsRemoveStaticLibrariesCalled).To(BeTrue())
					Expect(platform.PackageFileListPath).To(Equal(path.Join(dirProvider.EtcDir(), "static_libraries_list")))
				})

				It("does NOTHING if settings.env.bosh.remove_static_libraries is NOT set", func() {
					err := bootstrap()
					Expect(err).NotTo(HaveOccurred())
					Expect(platform.IsRemoveStaticLibrariesCalled).To(BeFalse())
				})

				It("does NOTHING if if settings.env.bosh.remove_static_libraries is true AND static_libraries_list does NOT exist", func() {
					settingsService.Settings.Env.Bosh.RemoveStaticLibraries = true
					err := bootstrap()
					Expect(err).NotTo(HaveOccurred())
					Expect(platform.IsRemoveStaticLibrariesCalled).To(BeFalse())
				})
			})

			Describe("checking persistent disks", func() {
				Context("managed persistent disk", func() {
					BeforeEach(func() {
						updateSettings := boshsettings.UpdateSettings{}
						updateSettingsBytes, err := json.Marshal(updateSettings)
						Expect(err).ToNot(HaveOccurred())

						updateSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "update_settings.json")
						platform.Fs.WriteFile(updateSettingsPath, updateSettingsBytes)
					})

					It("succesfully bootstraps", func() {
						err := bootstrap()
						Expect(err).ToNot(HaveOccurred())
					})

					Context("there is a single managed persistent disk attached", func() {
						BeforeEach(func() {
							settingsService.Settings.Disks = boshsettings.Disks{
								Persistent: map[string]interface{}{
									"vol-123": "/dev/sdb",
								},
							}
						})

						It("succesfully bootstraps", func() {
							err := bootstrap()
							Expect(err).ToNot(HaveOccurred())
						})
					})

					Context("there are multiple managed persistent disk attached", func() {
						BeforeEach(func() {
							settingsService.Settings.Disks = boshsettings.Disks{
								Persistent: map[string]interface{}{
									"vol-123": "/dev/sdb",
									"vol-456": "/dev/sdc",
								},
							}
						})

						It("returns an error", func() {
							err := bootstrap()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Unexpected disk attached"))
						})
					})

					Context("last mount information is present", func() {
						var managedDiskSettingsPath string

						BeforeEach(func() {
							diskCid := "i-am-a-disk-cid"

							managedDiskSettingsPath = filepath.Join(platform.GetDirProvider().BoshDir(), "managed_disk_settings.json")
							platform.Fs.WriteFile(managedDiskSettingsPath, []byte(diskCid))

							settingsService.Settings.Disks = boshsettings.Disks{
								Persistent: map[string]interface{}{
									"i-am-a-disk-cid": "/dev/sdb",
								},
							}
						})

						It("successfully bootstraps", func() {
							err := bootstrap()
							Expect(err).ToNot(HaveOccurred())
						})

						Context("when the last mount information cannot be read", func() {
							It("returns an error", func() {
								platform.Fs.RegisterReadFileError(managedDiskSettingsPath, errors.New("Oh noes!"))
								err := bootstrap()
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("Reading managed_disk_settings.json"))
							})
						})

						Context("attached disk's CID differs from last mounted CID", func() {
							BeforeEach(func() {
								diskCid := "i-am-a-different-cid"

								platform.Fs.WriteFile(managedDiskSettingsPath, []byte(diskCid))
							})

							It("returns an error", func() {
								err := bootstrap()
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("Attached disk disagrees with previous mount"))
							})
						})

						Context("when there are no attached disks", func() {
							BeforeEach(func() {
								settingsService.Settings.Disks = boshsettings.Disks{}
							})

							It("successfully bootstraps", func() {
								err := bootstrap()
								Expect(err).ToNot(HaveOccurred())
							})
						})
					})
				})

				Context("unmanaged persistent disk", func() {
					BeforeEach(func() {
						updateSettings := boshsettings.UpdateSettings{
							DiskAssociations: []boshsettings.DiskAssociation{
								boshsettings.DiskAssociation{
									Name:    "test-disk",
									DiskCID: "vol-123",
								},
								boshsettings.DiskAssociation{
									Name:    "test-disk-2",
									DiskCID: "vol-456",
								},
							},
						}

						updateSettingsBytes, err := json.Marshal(updateSettings)
						Expect(err).ToNot(HaveOccurred())

						updateSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "update_settings.json")
						platform.Fs.WriteFile(updateSettingsPath, updateSettingsBytes)

						settingsService.Settings.Disks = boshsettings.Disks{
							Persistent: map[string]interface{}{
								"vol-123": "/dev/sdb",
								"vol-456": "/dev/sdc",
							},
						}
					})

					It("succesfully bootstraps", func() {
						err := bootstrap()
						Expect(err).ToNot(HaveOccurred())
					})

					Context("a disk is not attached that should be", func() {
						BeforeEach(func() {
							settingsService.Settings.Disks = boshsettings.Disks{}
						})

						It("returns an error", func() {
							err := bootstrap()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Disk vol-123 is not attached"))
						})
					})

					Context("A disk is attached that shouldn't be", func() {
						BeforeEach(func() {
							settingsService.Settings.Disks = boshsettings.Disks{
								Persistent: map[string]interface{}{
									"vol-123": "/dev/sdb",
									"vol-456": "/dev/sdc",
									"vol-789": "/dev/sdd",
								},
							}
						})

						It("returns an error", func() {
							err := bootstrap()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Unexpected disk attached"))
						})
					})
				})

				Context("update_settings.json does not exist", func() {
					Context("there are multiple disks in the registry for this instance", func() {
						BeforeEach(func() {
							settingsService.Settings.Disks = boshsettings.Disks{
								Persistent: map[string]interface{}{
									"vol-123": "/dev/sdb",
									"vol-456": "/dev/sdc",
								},
							}
						})

						It("returns error", func() {
							err := bootstrap()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Unexpected disk attached"))
						})
					})

					Context("there are no disks in the registry for this instance", func() {
						It("succesfully bootstraps", func() {
							err := bootstrap()
							Expect(err).ToNot(HaveOccurred())
						})
					})
				})

				Context("when update_settings.json exists but cannot be read", func() {
					BeforeEach(func() {
						updateSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "update_settings.json")
						platform.Fs.WriteFile(updateSettingsPath, []byte(`{"persistent_disks":{"invalid":true`))
					})

					It("returns error", func() {
						platform.Fs.ReadFileError = errors.New("Oh noes!")

						err := bootstrap()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Reading update_settings.json"))
					})
				})

				Context("when unmarshalling update_settings fails", func() {
					BeforeEach(func() {
						updateSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "update_settings.json")
						platform.Fs.WriteFile(updateSettingsPath, []byte(`{"persistent_disks":{"invalid":true`))
					})

					It("returns wrapped error", func() {
						err := bootstrap()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Unmarshalling update_settings.json"))
					})
				})
			})

			Describe("Mount persistent disk", func() {
				Context("when there is no persistent disk", func() {
					It("does not try to mount ", func() {
						settingsService.Settings.Disks = boshsettings.Disks{
							Persistent: map[string]interface{}{},
						}

						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())
						Expect(platform.MountPersistentDiskSettings).To(Equal(boshsettings.DiskSettings{}))
						Expect(platform.MountPersistentDiskMountPoint).To(Equal(""))
					})
				})

				Context("when there is no drive specified by settings", func() {
					It("returns error", func() {
						settingsService.Settings.Disks = boshsettings.Disks{
							Persistent: map[string]interface{}{
								"vol-123": "/dev/not-exists",
							},
						}
						platform.SetIsPersistentDiskMountable(false, errors.New("Drive not exist!"))

						err := bootstrap()
						Expect(err).To(HaveOccurred())
						Expect(platform.MountPersistentDiskSettings).To(Equal(boshsettings.DiskSettings{}))
						Expect(platform.MountPersistentDiskMountPoint).To(Equal(""))
					})
				})

				Context("when there is no partition on drive specified by settings", func() {
					BeforeEach(func() {
						updateSettings := boshsettings.UpdateSettings{}
						updateSettingsBytes, err := json.Marshal(updateSettings)
						Expect(err).ToNot(HaveOccurred())

						updateSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "update_settings.json")
						platform.Fs.WriteFile(updateSettingsPath, updateSettingsBytes)
					})

					It("does not try to mount ", func() {
						settingsService.Settings.Disks = boshsettings.Disks{
							Persistent: map[string]interface{}{
								"vol-123": "/dev/valid",
							},
						}
						platform.SetIsPersistentDiskMountable(false, nil)

						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())
						Expect(platform.MountPersistentDiskSettings).To(Equal(boshsettings.DiskSettings{}))
						Expect(platform.MountPersistentDiskMountPoint).To(Equal(""))
					})
				})

				Context("when specified disk has partition", func() {
					BeforeEach(func() {
						updateSettings := boshsettings.UpdateSettings{}
						updateSettingsBytes, err := json.Marshal(updateSettings)
						Expect(err).ToNot(HaveOccurred())

						updateSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "update_settings.json")
						platform.Fs.WriteFile(updateSettingsPath, updateSettingsBytes)

						settingsService.Settings.Disks = boshsettings.Disks{
							Persistent: map[string]interface{}{
								"vol-123": map[string]interface{}{
									"volume_id": "2",
									"path":      "/dev/sdb",
								},
							},
						}
					})

					It("does not mount the disk", func() {
						platform.SetIsPersistentDiskMountable(true, nil)

						err := bootstrap()
						Expect(err).NotTo(HaveOccurred())
						Expect(platform.MountPersistentDiskSettings).To(Equal(boshsettings.DiskSettings{}))
						Expect(platform.MountPersistentDiskMountPoint).To(Equal(""))
					})

					Context("when last mounted cid information is present", func() {
						BeforeEach(func() {
							diskCid := "vol-123"

							managedDiskSettingsPath := filepath.Join(platform.GetDirProvider().BoshDir(), "managed_disk_settings.json")
							platform.Fs.WriteFile(managedDiskSettingsPath, []byte(diskCid))
						})

						It("mounts persistent disk", func() {
							platform.SetIsPersistentDiskMountable(true, nil)

							err := bootstrap()
							Expect(err).NotTo(HaveOccurred())
							Expect(platform.MountPersistentDiskSettings).To(Equal(boshsettings.DiskSettings{
								ID:       "vol-123",
								VolumeID: "2",
								Path:     "/dev/sdb",
							}))
							Expect(platform.MountPersistentDiskMountPoint).To(Equal(dirProvider.StoreDir()))
						})
					})
				})
			})
		})

		Describe("Network setup exercised by Run", func() {
			var (
				settingsJSON string

				fs                     *fakesys.FakeFileSystem
				platform               boshplatform.Platform
				boot                   Bootstrap
				defaultNetworkResolver boshsettings.DefaultNetworkResolver
				logger                 boshlog.Logger
				dirProvider            boshdirs.Provider

				interfaceAddrsProvider *fakeip.FakeInterfaceAddressesProvider
			)

			writeNetworkDevice := func(iface string, macAddress string, isPhysical bool) string {
				interfacePath := fmt.Sprintf("/sys/class/net/%s", iface)
				fs.WriteFile(interfacePath, []byte{})
				if isPhysical {
					fs.WriteFile(fmt.Sprintf("/sys/class/net/%s/device", iface), []byte{})
				}
				fs.WriteFileString(fmt.Sprintf("/sys/class/net/%s/address", iface), fmt.Sprintf("%s\n", macAddress))

				return interfacePath
			}

			stubInterfaces := func(interfaces [][]string) {
				var interfacePaths []string

				for _, iface := range interfaces {
					interfaceName := iface[0]
					interfaceMAC := iface[1]
					interfaceType := iface[2]
					isPhysical := interfaceType == "physical"
					interfacePaths = append(interfacePaths, writeNetworkDevice(interfaceName, interfaceMAC, isPhysical))
				}

				fs.SetGlob("/sys/class/net/*", interfacePaths)
			}

			BeforeEach(func() {
				fs = fakesys.NewFakeFileSystem()
				runner := fakesys.NewFakeCmdRunner()
				dirProvider = boshdirs.NewProvider("/var/vcap/bosh")

				linuxOptions := boshplatform.LinuxOptions{
					CreatePartitionIfNoEphemeralDisk: true,
				}

				logger = boshlog.NewLogger(boshlog.LevelNone)

				diskManager := fakedisk.NewFakeDiskManager()
				diskManager.FakeMountsSearcher.SearchMountsMounts = []boshdisk.Mount{
					{MountPoint: "/", PartitionPath: "rootfs"},
					{MountPoint: "/", PartitionPath: "/dev/vda1"},
				}

				// for the GrowRootFS call to findRootDevicePath
				runner.AddCmdResult(
					"readlink -f /dev/vda1",
					fakesys.FakeCmdResult{Stdout: "/dev/vda1"},
				)

				// for the createEphemeralPartitionsOnRootDevice call to findRootDevicePath
				runner.AddCmdResult(
					"readlink -f /dev/vda1",
					fakesys.FakeCmdResult{Stdout: "/dev/vda1"},
				)

				diskManager.FakeRootDevicePartitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = 1024 * 1024 * 1024

				udev := boshudev.NewConcreteUdevDevice(runner, logger)
				linuxCdrom := boshcdrom.NewLinuxCdrom("/dev/sr0", udev, runner)
				linuxCdutil := boshcdrom.NewCdUtil(dirProvider.SettingsDir(), fs, linuxCdrom, logger)

				compressor := boshcmd.NewTarballCompressor(runner, fs)
				copier := boshcmd.NewGenericCpCopier(fs, logger)

				sigarCollector := boshsigar.NewSigarStatsCollector(&sigar.ConcreteSigar{})

				vitalsService := boshvitals.NewService(sigarCollector, dirProvider)

				ipResolver := boship.NewResolver(boship.NetworkInterfaceToAddrsFunc)

				arping := bosharp.NewArping(runner, fs, logger, boshplatform.ArpIterations, boshplatform.ArpIterationDelay, boshplatform.ArpInterfaceCheckDelay)
				interfaceConfigurationCreator := boshnet.NewInterfaceConfigurationCreator(logger)

				interfaceAddrsProvider = &fakeip.FakeInterfaceAddressesProvider{}
				interfaceAddressesValidator := boship.NewInterfaceAddressesValidator(interfaceAddrsProvider)
				dnsValidator := boshnet.NewDNSValidator(fs)
				fs.WriteFileString("/etc/resolv.conf", "8.8.8.8 4.4.4.4")
				ubuntuNetManager := boshnet.NewUbuntuNetManager(fs, runner, ipResolver, interfaceConfigurationCreator, interfaceAddressesValidator, dnsValidator, arping, logger)

				ubuntuCertManager := boshcert.NewUbuntuCertManager(fs, runner, 1, logger)

				monitRetryable := boshplatform.NewMonitRetryable(runner)
				monitRetryStrategy := boshretry.NewAttemptRetryStrategy(10, 1*time.Second, monitRetryable, logger)

				devicePathResolver := devicepathresolver.NewIdentityDevicePathResolver()

				fakeUUIDGenerator := boshuuid.NewGenerator()
				routesSearcher := boshnet.NewRoutesSearcher(runner)
				defaultNetworkResolver = boshnet.NewDefaultNetworkResolver(routesSearcher, ipResolver)
				state, err := boshplatform.NewBootstrapState(fs, "/tmp/agent_state.json")
				Expect(err).NotTo(HaveOccurred())

				platform = boshplatform.NewLinuxPlatform(
					fs,
					runner,
					sigarCollector,
					compressor,
					copier,
					dirProvider,
					vitalsService,
					linuxCdutil,
					diskManager,
					ubuntuNetManager,
					ubuntuCertManager,
					monitRetryStrategy,
					devicePathResolver,
					state,
					linuxOptions,
					logger,
					defaultNetworkResolver,
					fakeUUIDGenerator,
					boshplatform.NewDelayedAuditLogger(fakeplatform.NewFakeAuditLoggerProvider(), logger),
				)
			})

			JustBeforeEach(func() {
				settingsPath := filepath.Join("bosh", "settings.json")

				var settings boshsettings.Settings
				json.Unmarshal([]byte(settingsJSON), &settings)

				settingsSource := fakeinf.FakeSettingsSource{
					PublicKey:     "123",
					SettingsValue: settings,
				}

				settingsService := boshsettings.NewService(
					platform.GetFs(),
					settingsPath,
					settingsSource,
					platform,
					logger,
				)

				boot = NewBootstrap(
					platform,
					dirProvider,
					settingsService,
					logger,
				)
			})

			Context("when a single network configuration is provided, with a MAC address", func() {
				BeforeEach(func() {
					settingsJSON = `{
					"networks": {
						"netA": {
							"default": ["dns", "gateway"],
							"ip": "2.2.2.2",
							"dns": [
								"8.8.8.8",
								"4.4.4.4"
							],
							"netmask": "255.255.255.0",
							"gateway": "2.2.2.0",
							"mac": "aa:bb:cc"
						}
					}
				}`
				})

				Context("and no physical network interfaces exist", func() {
					Context("and a single virtual network interface exists", func() {
						BeforeEach(func() {
							stubInterfaces([][]string{[]string{"lo", "aa:bb:cc", "virtual"}})
						})

						It("raises an error", func() {
							err := boot.Run()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Number of network settings '1' is greater than the number of network devices '0"))
						})
					})
				})

				Context("and a single physical network interface exists", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("and extra physical network interfaces exist", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}, []string{"eth1", "aa:bb:dd", "physical"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("and extra virtual network interfaces exist", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}, []string{"lo", "aa:bb:ee", "virtual"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).ToNot(HaveOccurred())
					})
				})
			})

			Context("when a single network configuration is provided, without a MAC address", func() {
				BeforeEach(func() {
					settingsJSON = `{
					"networks": {
						"netA": {
							"default": ["dns", "gateway"],
							"ip": "2.2.2.2",
							"dns": [
								"8.8.8.8",
								"4.4.4.4"
							],
							"netmask": "255.255.255.0",
							"gateway": "2.2.2.0"
						}
					}
				}`
				})

				Context("and no physical network interfaces exist", func() {
					Context("and a single virtual network interface exists", func() {
						BeforeEach(func() {
							stubInterfaces([][]string{[]string{"lo", "aa:bb:cc", "virtual"}})
						})

						It("raises an error", func() {
							err := boot.Run()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Number of network settings '1' is greater than the number of network devices '0"))
						})
					})
				})

				Context("and a single physical network interface exists", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("and extra physical network interfaces exist", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}, []string{"eth1", "aa:bb:dd", "physical"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("and an extra virtual network interface exists", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}, []string{"lo", "aa:bb:dd", "virtual"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})

			Context("when two network configurations are provided", func() {
				BeforeEach(func() {
					settingsJSON = `{
					"networks": {
						"netA": {
							"default": ["dns", "gateway"],
							"ip": "2.2.2.2",
							"dns": [
								"8.8.8.8",
								"4.4.4.4"
							],
							"netmask": "255.255.255.0",
							"gateway": "2.2.2.0",
							"mac": "aa:bb:cc"
						},
						"netB": {
							"default": ["dns", "gateway"],
							"ip": "3.3.3.3",
							"dns": [
								"8.8.8.8",
								"4.4.4.4"
							],
							"netmask": "255.255.255.0",
							"gateway": "3.3.3.0",
							"mac": ""
						}
					}
				}`
				})

				Context("and a single physical network interface exists", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}})
					})

					It("raises an error", func() {
						err := boot.Run()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Number of network settings '2' is greater than the number of network devices '1"))
					})
				})

				Context("and two physical network interfaces with matching MAC addresses exist", func() {
					BeforeEach(func() {
						stubInterfaces([][]string{[]string{"eth0", "aa:bb:cc", "physical"}, []string{"eth1", "aa:bb:dd", "physical"}})
						interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
							boship.NewSimpleInterfaceAddress("eth0", "2.2.2.2"),
							boship.NewSimpleInterfaceAddress("eth1", "3.3.3.3"),
						}
					})

					It("succeeds", func() {
						err := boot.Run()
						Expect(err).ToNot(HaveOccurred())
					})
				})
			})
		})
	})
}
