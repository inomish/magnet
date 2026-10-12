package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type xkbBinding struct {
	protocol.RiverXkbBindingV1Stub

	object protocol.RiverXkbBindingV1

	seat     *seat
	actionFn func()
}

func newXkbBinding(seat *seat, keysym uint32, mods uint32, fn func()) *xkbBinding {
	binding := &xkbBinding{
		object:   seat.windowManager.XkbBindingsV1.GetXkbBinding(seat.object, keysym, mods),
		seat:     seat,
		actionFn: fn,
	}

	binding.object.SetUserData(binding)

	return binding
}

func (binding *xkbBinding) HandleRiverXkbBindingV1Pressed(ctx context.Context) {
	binding.seat.pendingAction = binding.actionFn
}
