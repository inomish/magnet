package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type output struct {
	protocol.RiverOutputV1Stub
	object  protocol.RiverOutputV1
	removed bool
}

func newOutput(object protocol.RiverOutputV1) *output {
	output := &output{
		object: object,
	}

	object.SetUserData(output)

	return output
}

func (o *output) HandleRiverOutputV1Removed(ctx context.Context) {
	o.removed = true
}

func (o *output) maybeDestroy() bool {
	if !o.removed {
		return false
	}

	o.object.Destroy()

	return true
}
