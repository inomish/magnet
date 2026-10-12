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

	outputs []*output
	windows []*window
	seats   []*seat
}

func (windowManager *WindowManager) HandleWlRegistryGlobal(
	ctx context.Context,
	name uint32,
	iface string,
	version uint32,
) {
	switch iface {
	case protocol.RiverWindowManagerV1Name:
		if version >= 4 {
			windowManager.WindowManagerV1 = protocol.As[protocol.RiverWindowManagerV1](windowManager.Registry.Bind(name, iface, 4))
		}
	case protocol.RiverXkbBindingsV1Name:
		windowManager.XkbBindingsV1 = protocol.As[protocol.RiverXkbBindingsV1](windowManager.Registry.Bind(name, iface, 1))
	}
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1Unavailable(ctx context.Context) {
	windowManager.Done = true
	windowManager.Err = errors.New("another window manager is already running")
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1Finished(ctx context.Context) {
	windowManager.Done = true
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1Output(
	ctx context.Context,
	id protocol.RiverOutputV1,
) {
	windowManager.outputs = append(windowManager.outputs, newOutput(id))
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1Window(
	ctx context.Context,
	id protocol.RiverWindowV1,
) {
	windowManager.windows = append(windowManager.windows, newWindow(id))
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1Seat(ctx context.Context, id protocol.RiverSeatV1) {
	windowManager.seats = append(windowManager.seats, newSeat(id, windowManager))
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1ManageStart(ctx context.Context) {
	windowManager.outputs = slices.DeleteFunc(windowManager.outputs, (*output).maybeDestroy)
	windowManager.windows = slices.DeleteFunc(windowManager.windows, (*window).maybeDestroy)
	windowManager.seats = slices.DeleteFunc(windowManager.seats, (*seat).maybeDestroy)

	for _, window := range windowManager.windows {
		window.manage()
	}
	for _, seat := range windowManager.seats {
		seat.manage()
	}

	windowManager.WindowManagerV1.ManageFinish()
}

func (windowManager *WindowManager) HandleRiverWindowManagerV1RenderStart(ctx context.Context) {
	windowManager.WindowManagerV1.RenderFinish()
}
