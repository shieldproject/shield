package settings_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/matchers"
	"github.com/cloudfoundry/bosh-agent/platform/disk"
	. "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("Settings", func() {
	var settings Settings

	Describe("PersistentDiskSettings", func() {
		Context("when the disk settings are hash", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Persistent: map[string]interface{}{
							"fake-disk-id": map[string]interface{}{
								"volume_id":      "fake-disk-volume-id",
								"id":             "fake-disk-device-id",
								"path":           "fake-disk-path",
								"lun":            "fake-disk-lun",
								"host_device_id": "fake-disk-host-device-id",
							},
						},
					},
				}
			})

			It("returns disk settings", func() {
				diskSettings, found := settings.PersistentDiskSettings("fake-disk-id")
				Expect(found).To(BeTrue())
				Expect(diskSettings).To(Equal(DiskSettings{
					ID:           "fake-disk-id",
					DeviceID:     "fake-disk-device-id",
					VolumeID:     "fake-disk-volume-id",
					Path:         "fake-disk-path",
					Lun:          "fake-disk-lun",
					HostDeviceID: "fake-disk-host-device-id",
				}))
			})

			Context("when disk with requested disk ID is not present", func() {
				It("returns false", func() {
					diskSettings, found := settings.PersistentDiskSettings("fake-non-existent-disk-id")
					Expect(found).To(BeFalse())
					Expect(diskSettings).To(Equal(DiskSettings{}))
				})
			})

			Context("when Env is provided", func() {
				It("gets filesystem type from env", func() {
					settingsJSON := `{"env": {"persistent_disk_fs": "xfs"}}`

					err := json.Unmarshal([]byte(settingsJSON), &settings)
					Expect(err).NotTo(HaveOccurred())
					diskSettings, _ := settings.PersistentDiskSettings("fake-disk-id")
					Expect(settings.Env.PersistentDiskFS).To(Equal(disk.FileSystemXFS))
					Expect(diskSettings).To(Equal(DiskSettings{
						ID:             "fake-disk-id",
						DeviceID:       "fake-disk-device-id",
						VolumeID:       "fake-disk-volume-id",
						Path:           "fake-disk-path",
						Lun:            "fake-disk-lun",
						HostDeviceID:   "fake-disk-host-device-id",
						FileSystemType: "xfs",
					}))
				})

				It("does not crash if env does not have a filesystem type", func() {
					settingsJSON := `{"env": {"bosh": {"password": "secret"}}}`

					err := json.Unmarshal([]byte(settingsJSON), &settings)
					Expect(err).NotTo(HaveOccurred())
					diskSettings, _ := settings.PersistentDiskSettings("fake-disk-id")
					Expect(settings.Env.PersistentDiskFS).To(Equal(disk.FileSystemDefault))
					Expect(diskSettings).To(Equal(DiskSettings{
						ID:           "fake-disk-id",
						DeviceID:     "fake-disk-device-id",
						VolumeID:     "fake-disk-volume-id",
						Path:         "fake-disk-path",
						Lun:          "fake-disk-lun",
						HostDeviceID: "fake-disk-host-device-id",
					}))
				})

				It("does not crash if env has a bad fs", func() {
					settingsJSON := `{"env": {"persistent_disk_fs": "blahblah"}}`

					err := json.Unmarshal([]byte(settingsJSON), &settings)
					Expect(err).NotTo(HaveOccurred())
					diskSettings, _ := settings.PersistentDiskSettings("fake-disk-id")
					Expect(settings.Env.PersistentDiskFS).To(Equal(disk.FileSystemType("blahblah")))
					Expect(diskSettings).To(Equal(DiskSettings{
						ID:             "fake-disk-id",
						DeviceID:       "fake-disk-device-id",
						VolumeID:       "fake-disk-volume-id",
						Path:           "fake-disk-path",
						Lun:            "fake-disk-lun",
						HostDeviceID:   "fake-disk-host-device-id",
						FileSystemType: disk.FileSystemType("blahblah"),
					}))
				})
			})
		})

		Context("when the disk settings is a string", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Persistent: map[string]interface{}{
							"fake-disk-id": "fake-disk-value",
						},
					},
				}
			})

			It("converts it to disk settings", func() {
				diskSettings, found := settings.PersistentDiskSettings("fake-disk-id")
				Expect(found).To(BeTrue())
				Expect(diskSettings).To(Equal(DiskSettings{
					ID:       "fake-disk-id",
					VolumeID: "fake-disk-value",
					Path:     "fake-disk-value",
				}))
			})

			Context("when disk with requested disk ID is not present", func() {
				It("returns false", func() {
					diskSettings, found := settings.PersistentDiskSettings("fake-non-existent-disk-id")
					Expect(found).To(BeFalse())
					Expect(diskSettings).To(Equal(DiskSettings{}))
				})
			})
		})

		Context("when DeviceID is not provided", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Persistent: map[string]interface{}{
							"fake-disk-id": map[string]interface{}{
								"volume_id":      "fake-disk-volume-id",
								"path":           "fake-disk-path",
								"lun":            "fake-disk-lun",
								"host_device_id": "fake-disk-host-device-id",
							},
						},
					},
				}
			})

			It("does not set id", func() {
				diskSettings, found := settings.PersistentDiskSettings("fake-disk-id")
				Expect(found).To(BeTrue())
				Expect(diskSettings).To(Equal(DiskSettings{
					ID:           "fake-disk-id",
					VolumeID:     "fake-disk-volume-id",
					Path:         "fake-disk-path",
					Lun:          "fake-disk-lun",
					HostDeviceID: "fake-disk-host-device-id",
				}))
			})
		})

		Context("when volume ID is not provided", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Persistent: map[string]interface{}{
							"fake-disk-id": map[string]interface{}{
								"id":             "fake-disk-device-id",
								"path":           "fake-disk-path",
								"lun":            "fake-disk-lun",
								"host_device_id": "fake-disk-host-device-id",
							},
						},
					},
				}
			})

			It("does not set id", func() {
				diskSettings, found := settings.PersistentDiskSettings("fake-disk-id")
				Expect(found).To(BeTrue())
				Expect(diskSettings).To(Equal(DiskSettings{
					ID:           "fake-disk-id",
					DeviceID:     "fake-disk-device-id",
					Path:         "fake-disk-path",
					Lun:          "fake-disk-lun",
					HostDeviceID: "fake-disk-host-device-id",
				}))
			})
		})

		Context("when path is not provided", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Persistent: map[string]interface{}{
							"fake-disk-id": map[string]interface{}{
								"volume_id":      "fake-disk-volume-id",
								"lun":            "fake-disk-lun",
								"host_device_id": "fake-disk-host-device-id",
							},
						},
					},
				}
			})

			It("does not set path", func() {
				diskSettings, found := settings.PersistentDiskSettings("fake-disk-id")
				Expect(found).To(BeTrue())
				Expect(diskSettings).To(Equal(DiskSettings{
					ID:           "fake-disk-id",
					VolumeID:     "fake-disk-volume-id",
					Lun:          "fake-disk-lun",
					HostDeviceID: "fake-disk-host-device-id",
				}))
			})
		})

		Context("when only (lun, host_device_id) are provided", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Persistent: map[string]interface{}{
							"fake-disk-id": map[string]interface{}{
								"lun":            "fake-disk-lun",
								"host_device_id": "fake-disk-host-device-id",
							},
						},
					},
				}
			})

			It("does not set path", func() {
				diskSettings, found := settings.PersistentDiskSettings("fake-disk-id")
				Expect(found).To(BeTrue())
				Expect(diskSettings).To(Equal(DiskSettings{
					ID:           "fake-disk-id",
					Lun:          "fake-disk-lun",
					HostDeviceID: "fake-disk-host-device-id",
				}))
			})
		})
	})

	Describe("EphemeralDiskSettings", func() {
		Context("when the disk settings are a string", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Ephemeral: "fake-disk-value",
					},
				}
			})

			It("converts disk settings", func() {
				Expect(settings.EphemeralDiskSettings()).To(Equal(DiskSettings{
					VolumeID: "fake-disk-value",
					Path:     "fake-disk-value",
				}))
			})
		})

		Context("when the disk settings are a hash", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Ephemeral: map[string]interface{}{
							"id":             "fake-disk-device-id",
							"volume_id":      "fake-disk-volume-id",
							"path":           "fake-disk-path",
							"lun":            "fake-disk-lun",
							"host_device_id": "fake-disk-host-device-id",
						},
					},
				}
			})

			It("converts disk settings", func() {
				Expect(settings.EphemeralDiskSettings()).To(Equal(DiskSettings{
					DeviceID:     "fake-disk-device-id",
					VolumeID:     "fake-disk-volume-id",
					Path:         "fake-disk-path",
					Lun:          "fake-disk-lun",
					HostDeviceID: "fake-disk-host-device-id",
				}))
			})
		})

		Context("when path is not provided", func() {
			BeforeEach(func() {
				settings = Settings{
					Disks: Disks{
						Ephemeral: map[string]interface{}{
							"id":        "fake-disk-device-id",
							"volume_id": "fake-disk-volume-id",
						},
					},
				}
			})

			It("does not set path", func() {
				Expect(settings.EphemeralDiskSettings()).To(Equal(DiskSettings{
					DeviceID: "fake-disk-device-id",
					VolumeID: "fake-disk-volume-id",
				}))
			})
		})
	})

	Describe("DefaultNetworkFor", func() {
		Context("when networks is empty", func() {
			It("returns found=false", func() {
				networks := Networks{}
				_, found := networks.DefaultNetworkFor("dns")
				Expect(found).To(BeFalse())
			})
		})

		Context("with a single network", func() {
			It("returns that network (found=true)", func() {
				networks := Networks{
					"first": Network{
						DNS: []string{"xx.xx.xx.xx"},
					},
				}

				network, found := networks.DefaultNetworkFor("dns")
				Expect(found).To(BeTrue())
				Expect(network).To(Equal(networks["first"]))
			})
		})

		Context("with multiple networks and default is found for dns", func() {
			It("returns the network marked default (found=true)", func() {
				networks := Networks{
					"first": Network{
						Default: []string{},
						DNS:     []string{"aa.aa.aa.aa"},
					},
					"second": Network{
						Default: []string{"something-else", "dns"},
						DNS:     []string{"xx.xx.xx.xx", "yy.yy.yy.yy", "zz.zz.zz.zz"},
					},
					"third": Network{
						Default: []string{},
						DNS:     []string{"aa.aa.aa.aa"},
					},
				}

				settings, found := networks.DefaultNetworkFor("dns")
				Expect(found).To(BeTrue())
				Expect(settings).To(Equal(networks["second"]))
			})
		})

		Context("with multiple networks and default is not found", func() {
			It("returns found=false", func() {
				networks := Networks{
					"first": Network{
						Default: []string{"foo"},
						DNS:     []string{"xx.xx.xx.xx", "yy.yy.yy.yy", "zz.zz.zz.zz"},
					},
					"second": Network{
						Default: []string{},
						DNS:     []string{"aa.aa.aa.aa"},
					},
				}

				_, found := networks.DefaultNetworkFor("dns")
				Expect(found).To(BeFalse())
			})
		})

		Context("with multiple networks marked as default", func() {
			It("returns one of them", func() {
				networks := Networks{
					"first": Network{
						Default: []string{"dns"},
						DNS:     []string{"xx.xx.xx.xx", "yy.yy.yy.yy", "zz.zz.zz.zz"},
					},
					"second": Network{
						Default: []string{"dns"},
						DNS:     []string{"aa.aa.aa.aa"},
					},
					"third": Network{
						DNS: []string{"bb.bb.bb.bb"},
					},
				}

				for i := 0; i < 100; i++ {
					settings, found := networks.DefaultNetworkFor("dns")
					Expect(found).To(BeTrue())
					Expect(settings).Should(MatchOneOf(networks["first"], networks["second"]))
				}
			})
		})
	})

	Describe("DefaultIP", func() {
		It("with two networks", func() {
			networks := Networks{
				"bosh": Network{
					IP: "xx.xx.xx.xx",
				},
				"vip": Network{
					IP: "aa.aa.aa.aa",
				},
			}

			ip, found := networks.DefaultIP()
			Expect(found).To(BeTrue())
			Expect(ip).To(MatchOneOf("xx.xx.xx.xx", "aa.aa.aa.aa"))
		})

		It("with two networks only with defaults", func() {
			networks := Networks{
				"bosh": Network{
					IP: "xx.xx.xx.xx",
				},
				"vip": Network{
					IP:      "aa.aa.aa.aa",
					Default: []string{"dns"},
				},
			}

			ip, found := networks.DefaultIP()
			Expect(found).To(BeTrue())
			Expect(ip).To(Equal("aa.aa.aa.aa"))
		})

		It("when none specified", func() {
			networks := Networks{
				"bosh": Network{},
				"vip": Network{
					Default: []string{"dns"},
				},
			}

			_, found := networks.DefaultIP()
			Expect(found).To(BeFalse())
		})
	})

	It("allows different types for blobstore option values", func() {
		settingsJSON := `{"blobstore":{"options":{"string":"value", "int":443, "bool":true, "map":{}}}}`

		err := json.Unmarshal([]byte(settingsJSON), &settings)
		Expect(err).NotTo(HaveOccurred())
		Expect(settings.Blobstore.Options).To(Equal(map[string]interface{}{
			"string": "value",
			"int":    443.0,
			"bool":   true,
			"map":    map[string]interface{}{},
		}))
	})

	Describe("Snake Case Settings", func() {
		var expectSnakeCaseKeys func(map[string]interface{})

		expectSnakeCaseKeys = func(value map[string]interface{}) {
			for k, v := range value {
				Expect(k).To(MatchRegexp("\\A[a-z0-9_]+\\z"))

				tv, isMap := v.(map[string]interface{})
				if isMap {
					expectSnakeCaseKeys(tv)
				}
			}
		}

		It("marshals into JSON in snake case to stay consistent with CPI agent env formatting", func() {
			settingsJSON, err := json.Marshal(settings)
			Expect(err).NotTo(HaveOccurred())

			var settingsMap map[string]interface{}
			err = json.Unmarshal(settingsJSON, &settingsMap)
			Expect(err).NotTo(HaveOccurred())
			expectSnakeCaseKeys(settingsMap)
		})
	})

	Describe("Network", func() {
		var network Network
		BeforeEach(func() {
			network = Network{}
		})

		Describe("IsDHCP", func() {
			Context("when network is VIP", func() {
				BeforeEach(func() {
					network.Type = NetworkTypeVIP
				})

				It("returns false", func() {
					Expect(network.IsDHCP()).To(BeFalse())
				})
			})

			Context("when network is Dynamic", func() {
				BeforeEach(func() {
					network.Type = NetworkTypeDynamic
				})

				It("returns true", func() {
					Expect(network.IsDHCP()).To(BeTrue())
				})
			})

			Context("when IP is not set", func() {
				BeforeEach(func() {
					network.Netmask = "255.255.255.0"
				})

				It("returns true", func() {
					Expect(network.IsDHCP()).To(BeTrue())
				})
			})

			Context("when Netmask is not set", func() {
				BeforeEach(func() {
					network.IP = "127.0.0.5"
				})

				It("returns true", func() {
					Expect(network.IsDHCP()).To(BeTrue())
				})
			})

			Context("when IP and Netmask are set", func() {
				BeforeEach(func() {
					network.IP = "127.0.0.5"
					network.Netmask = "255.255.255.0"
				})

				It("returns false", func() {
					Expect(network.IsDHCP()).To(BeFalse())
				})
			})

			Context("when network was previously resolved via DHCP", func() {
				BeforeEach(func() {
					network.Resolved = true
				})

				It("returns true", func() {
					Expect(network.IsDHCP()).To(BeTrue())
				})
			})

			Context("when UseDHCP is true", func() {
				BeforeEach(func() {
					network.UseDHCP = true
					network.IP = "127.0.0.5"
					network.Netmask = "255.255.255.0"
				})

				It("returns true", func() {
					Expect(network.IsDHCP()).To(BeTrue())
				})
			})
		})
	})

	Describe("Networks", func() {
		network1 := Network{}
		network2 := Network{}
		network3 := Network{}
		networks := Networks{}

		BeforeEach(func() {
			network1.Type = NetworkTypeVIP
			network2.Preconfigured = true
			network3.Preconfigured = false
		})

		Describe("IsPreconfigured", func() {
			Context("with VIP and all preconfigured networks", func() {
				BeforeEach(func() {
					networks = Networks{
						"first":  network1,
						"second": network2,
					}
				})

				It("returns true", func() {
					Expect(networks.IsPreconfigured()).To(BeTrue())
				})
			})

			Context("with VIP and NOT all preconfigured networks", func() {
				BeforeEach(func() {
					networks = Networks{
						"first":  network1,
						"second": network2,
						"third":  network3,
					}
				})

				It("returns false", func() {
					Expect(networks.IsPreconfigured()).To(BeFalse())
				})
			})

			Context("with NO VIP and all preconfigured networks", func() {
				BeforeEach(func() {
					networks = Networks{
						"first": network2,
					}
				})

				It("returns true", func() {
					Expect(networks.IsPreconfigured()).To(BeTrue())
				})
			})

			Context("with NO VIP and NOT all preconfigured networks", func() {
				BeforeEach(func() {
					networks = Networks{
						"first":  network2,
						"second": network3,
					}
				})

				It("returns false", func() {
					Expect(networks.IsPreconfigured()).To(BeFalse())
				})
			})
		})
	})

	Describe("Env", func() {
		It("unmarshal env value correctly", func() {
			var env Env
			envJSON := `{"bosh": {"password": "fake-password", "keep_root_password": false, "remove_dev_tools": true}}`

			err := json.Unmarshal([]byte(envJSON), &env)
			Expect(err).NotTo(HaveOccurred())
			Expect(env.GetPassword()).To(Equal("fake-password"))
			Expect(env.GetKeepRootPassword()).To(BeFalse())
			Expect(env.GetRemoveDevTools()).To(BeTrue())
		})
	})
})
