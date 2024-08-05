package flowcontext

import (
	"github.com/Hoosat-Oy/HTND/infrastructure/network/addressmanager"
)

// AddressManager returns the address manager associated to the flow context.
func (f *FlowContext) AddressManager() *addressmanager.AddressManager {
	return f.addressManager
}
