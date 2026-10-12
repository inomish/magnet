package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type XkbBinding struct {
	protocol.RiverXkbBindingV1Stub

	Object protocol.RiverXkbBindingV1

	Seat     *Seat
	ActionFn func()
}

func NewXkbBinding(seat *Seat, keysym uint32, mods uint32, fn func()) *XkbBinding {
	binding := &XkbBinding{
		Object:   seat.WindowManager.XkbBindingsV1.GetXkbBinding(seat.Object, keysym, mods),
		Seat:     seat,
		ActionFn: fn,
	}

	binding.Object.SetUserData(binding)

	return binding
}

func (p *XkbBinding) HandleRiverXkbBindingV1Pressed(ctx context.Context) {
	p.Seat.PendingAction = p.ActionFn
}

type PointerBinding struct {
	protocol.RiverPointerBindingV1Stub

	Object protocol.RiverPointerBindingV1

	Seat     *Seat
	ActionFn func()
}

func NewPointerBinding(seat *Seat, button uint32, mods uint32, fn func()) *PointerBinding {
	binding := &PointerBinding{
		Object:   seat.Object.GetPointerBinding(button, mods),
		Seat:     seat,
		ActionFn: fn,
	}

	binding.Object.SetUserData(binding)

	return binding
}

func (p *PointerBinding) HandleRiverPointerBindingV1Pressed(ctx context.Context) {
	p.Seat.PendingAction = p.ActionFn
}
