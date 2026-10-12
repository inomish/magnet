package windowmanager

import (
	"context"

	"github.com/inomish/magnet/generated/protocol"
)

type Output struct {
	protocol.RiverOutputV1Stub
	Object  protocol.RiverOutputV1
	Removed bool
}

func NewOutput(object protocol.RiverOutputV1) *Output {
	output := &Output{
		Object: object,
	}

	object.SetUserData(output)

	return output
}

func (o *Output) HandleRiverOutputV1Removed(ctx context.Context) {
	o.Removed = true
}

func (o *Output) MaybeDestroy() bool {
	if !o.Removed {
		return false
	}

	o.Object.Destroy()

	return true
}
