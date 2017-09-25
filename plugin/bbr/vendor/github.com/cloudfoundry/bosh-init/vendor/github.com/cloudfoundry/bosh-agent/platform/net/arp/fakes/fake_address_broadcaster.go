package fakes

import (
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
)

type FakeAddressBroadcaster struct {
	BroadcastMACAddressesAddresses []boship.InterfaceAddress
}

func (b *FakeAddressBroadcaster) BroadcastMACAddresses(addresses []boship.InterfaceAddress) {
	b.BroadcastMACAddressesAddresses = addresses
}
