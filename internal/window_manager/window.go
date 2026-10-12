package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type window struct {
	protocol.RiverWindowV1Stub

	object protocol.RiverWindowV1
	node   protocol.RiverNodeV1

	x, y          int32
	width, height int32

	isNew  bool
	closed bool

	pointerMoveRequested *seat

	pointerResizeRequested      *seat
	pointerResizeRequestedEdges uint32
}

func (w *window) setPosition(x, y int32) {
	w.node.SetPosition(x, y)
	w.x, w.y = x, y
}

func (w *window) HandleRiverWindowV1Closed(ctx context.Context) {
	w.closed = true
}

func (w *window) HandleRiverWindowV1Dimensions(ctx context.Context, width int32, height int32) {
	w.width, w.height = width, height
}

func (w *window) HandleRiverWindowV1PointerMoveRequested(
	ctx context.Context,
	riverSeat protocol.RiverSeatV1,
) {
	w.pointerMoveRequested = riverSeat.UserData().(*seat)
}

func (w *window) HandleRiverWindowV1PointerResizeRequested(
	ctx context.Context,
	riverSeat protocol.RiverSeatV1,
	edges uint32,
) {
	w.pointerResizeRequested = riverSeat.UserData().(*seat)
	w.pointerResizeRequestedEdges = edges
}

func (w *window) maybeDestroy() bool {
	if !w.closed {
		return false
	}

	w.node.Destroy()
	w.object.Destroy()
	return true
}

func (w *window) manage() {
	if w.isNew {
		w.isNew = false
		w.setPosition(0, 0)
		w.object.ProposeDimensions(0, 0)
	}

	if w.pointerMoveRequested != nil {
		w.pointerMoveRequested.pointerMove(w)
		w.pointerMoveRequested = nil
	}

	if w.pointerResizeRequested != nil {
		w.pointerResizeRequested.pointerResize(w, w.pointerResizeRequestedEdges)
		w.pointerResizeRequested = nil
	}
}

func newWindow(object protocol.RiverWindowV1) *window {
	window := &window{
		object: object,
		node:   object.GetNode(),
		isNew:  true,
	}

	object.SetUserData(window)
	return window
}
