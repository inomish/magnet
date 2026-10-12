package windowmanager

import (
	"context"
	"errors"
	"slices"

	"github.com/inomish/magnet/generated/protocol"
)

type WindowManager struct {
	protocol.WlRegistryStub
	protocol.RiverWindowManagerV1Stub

	Registry        protocol.WlRegistry
	WindowManagerV1 protocol.RiverWindowManagerV1
	XkbBindingsV1   protocol.RiverXkbBindingsV1

	Done bool
	Err  error

	Outputs []*Output
	Windows []*Window
	Seats   []*Seat
}

func (wm *WindowManager) HandleWlRegistryGlobal(
	ctx context.Context,
	name uint32,
	iface string,
	version uint32,
) {
	switch iface {
	case protocol.RiverWindowManagerV1Name:
		if version >= 4 {
			wm.WindowManagerV1 = protocol.As[protocol.RiverWindowManagerV1](wm.Registry.Bind(name, iface, 4))
		}
	case protocol.RiverXkbBindingsV1Name:
		wm.XkbBindingsV1 = protocol.As[protocol.RiverXkbBindingsV1](wm.Registry.Bind(name, iface, 1))
	}
}

func (wm *WindowManager) HandleRiverWindowManagerV1Unavailable(ctx context.Context) {
	wm.Done = true
	wm.Err = errors.New("another window manager is already running")
}

func (wm *WindowManager) HandleRiverWindowManagerV1Finished(ctx context.Context) {
	wm.Done = true
}

func (wm *WindowManager) HandleRiverWindowManagerV1Output(
	ctx context.Context,
	id protocol.RiverOutputV1,
) {
	wm.Outputs = append(wm.Outputs, NewOutput(id))
}

func (wm *WindowManager) HandleRiverWindowManagerV1Window(
	ctx context.Context,
	id protocol.RiverWindowV1,
) {
	wm.Windows = append(wm.Windows, NewWindow(id))
}

func (wm *WindowManager) HandleRiverWindowManagerV1Seat(ctx context.Context, id protocol.RiverSeatV1) {
	wm.Seats = append(wm.Seats, NewSeat(id, wm))
}

func (wm *WindowManager) HandleRiverWindowManagerV1ManageStart(ctx context.Context) {
	wm.Outputs = slices.DeleteFunc(wm.Outputs, (*Output).MaybeDestroy)
	wm.Windows = slices.DeleteFunc(wm.Windows, (*Window).MaybeDestroy)
	wm.Seats = slices.DeleteFunc(wm.Seats, (*Seat).MaybeDestroy)

	for _, w := range wm.Windows {
		w.Manage()
	}
	for _, s := range wm.Seats {
		s.Manage()
	}

	wm.WindowManagerV1.ManageFinish()
}

func (wm *WindowManager) HandleRiverWindowManagerV1RenderStart(ctx context.Context) {
	for _, s := range wm.Seats {
		s.Render()
	}

	wm.WindowManagerV1.RenderFinish()
}
