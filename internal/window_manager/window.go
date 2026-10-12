package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type Window struct {
	protocol.RiverWindowV1Stub

	Object protocol.RiverWindowV1
	Node   protocol.RiverNodeV1

	X, Y          int32
	Width, Height int32

	New    bool
	Closed bool

	PointerMoveRequested *Seat

	PointerResizeRequested      *Seat
	PointerResizeRequestedEdges uint32
}

func (w *Window) SetPosition(x, y int32) {
	w.Node.SetPosition(x, y)
	w.X, w.Y = x, y
}

func (w *Window) HandleRiverWindowV1Closed(ctx context.Context) {
	w.Closed = true
}

func (w *Window) HandleRiverWindowV1Dimensions(ctx context.Context, width int32, height int32) {
	w.Width, w.Height = width, height
}

func (w *Window) HandleRiverWindowV1PointerMoveRequested(
	ctx context.Context,
	seat protocol.RiverSeatV1,
) {
	w.PointerMoveRequested = seat.UserData().(*Seat)
}

func (w *Window) HandleRiverWindowV1PointerResizeRequested(
	ctx context.Context,
	seat protocol.RiverSeatV1,
	edges uint32,
) {
	w.PointerResizeRequested = seat.UserData().(*Seat)
	w.PointerResizeRequestedEdges = edges
}

func (w *Window) MaybeDestroy() bool {
	if !w.Closed {
		return false
	}

	w.Node.Destroy()
	w.Object.Destroy()
	return true
}

func (w *Window) Manage() {
	if w.New {
		w.New = false
		w.SetPosition(0, 0)
		w.Object.ProposeDimensions(0, 0)
	}

	if w.PointerMoveRequested != nil {
		w.PointerMoveRequested.PointerMove(w)
		w.PointerMoveRequested = nil
	}

	if w.PointerResizeRequested != nil {
		w.PointerResizeRequested.PointerResize(w, w.PointerResizeRequestedEdges)
		w.PointerResizeRequested = nil
	}
}

func NewWindow(object protocol.RiverWindowV1) *Window {
	window := &Window{
		Object: object,
		Node:   object.GetNode(),
		New:    true,
	}

	object.SetUserData(window)
	return window
}
