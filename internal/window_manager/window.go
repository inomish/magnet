package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type window struct {
	protocol.RiverWindowV1Stub

	object protocol.RiverWindowV1
	node   protocol.RiverNodeV1

	decorationHint uint32

	isNew  bool
	closed bool
}

func (w *window) HandleRiverWindowV1Closed(ctx context.Context) {
	w.closed = true
}

func (w *window) HandleRiverWindowV1DecorationHint(ctx context.Context, hint uint32) {
	w.decorationHint = hint
}

func (w *window) setPosition(x, y int32) {
	w.node.SetPosition(x, y)
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

	if w.decorationHint == protocol.RiverWindowV1DecorationHintOnlySupportsCsd {
		w.object.UseCsd()
	} else {
		w.object.UseSsd()
	}
}

func newWindow(object protocol.RiverWindowV1) *window {
	window := &window{
		object:         object,
		node:           object.GetNode(),
		decorationHint: protocol.RiverWindowV1DecorationHintNoPreference,
		isNew:          true,
	}

	object.SetUserData(window)

	return window
}
