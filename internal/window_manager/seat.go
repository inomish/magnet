package windowmanager

import (
	"context"
	"log"
	"os/exec"
	"slices"

	"github.com/inomish/magnet/generated/protocol"
)

type seat struct {
	protocol.RiverSeatV1Stub

	windowManager *WindowManager

	object      protocol.RiverSeatV1
	xkbBindings []*xkbBinding

	focused    *window
	interacted *window

	pendingAction func()

	isNew   bool
	removed bool
}

func newSeat(object protocol.RiverSeatV1, windowManager *WindowManager) *seat {
	const (
		// See xkbcommon-keysyms.h
		spaceSym = 0x0020
		nSym     = 0x006e
		qSym     = 0x0071
		escSym   = 0xff1b
	)

	const super = protocol.RiverSeatV1ModifiersMod1

	seat := &seat{
		windowManager: windowManager,
		object:        object,
		isNew:         true,
	}

	seat.xkbBindings = []*xkbBinding{
		newXkbBinding(seat, spaceSym, super, func() {
			cmd := exec.Command("kitty")

			if err := cmd.Start(); err != nil {
				log.Printf("Failed to launch foot: %v", err)
				return
			}
			go func() { _, _ = cmd.Process.Wait() }()
		}),
		newXkbBinding(seat, nSym, super, func() {
			if len(seat.windowManager.windows) > 0 {
				seat.focus(seat.windowManager.windows[0])
			}
		}),
		newXkbBinding(seat, qSym, super, func() {
			if seat.focused != nil {
				seat.focused.object.Close()
			}
		}),
		newXkbBinding(seat, escSym, super, func() {
			seat.windowManager.WindowManagerV1.ExitSession()
		}),
	}

	seat.object.SetUserData(seat)

	return seat
}

func (s *seat) focus(window *window) {
	windowManager := s.windowManager

	if window == nil && len(windowManager.windows) > 0 {
		window = windowManager.windows[len(windowManager.windows)-1]
	}

	if s.focused == window {
		return
	}

	if window != nil {
		s.object.FocusWindow(window.object)

		// Raise to top
		index := slices.Index(windowManager.windows, window)
		windowManager.windows = append(windowManager.windows[:index], windowManager.windows[index+1:]...)
		windowManager.windows = append(windowManager.windows, window)

		window.node.PlaceTop()
	} else {
		s.object.ClearFocus()
	}

	s.focused = window
}

func (s *seat) HandleRiverSeatV1Removed(ctx context.Context) {
	s.removed = true
}

func (s *seat) HandleRiverSeatV1WindowInteraction(ctx context.Context, riverWindow protocol.RiverWindowV1) {
	s.interacted = riverWindow.UserData().(*window)
}

func (s *seat) maybeDestroy() bool {
	if !s.removed {
		return false
	}

	for _, binding := range s.xkbBindings {
		binding.object.Destroy()
	}

	s.object.Destroy()

	return true
}

func (s *seat) manage() {
	if s.isNew {
		s.isNew = false
		for _, binding := range s.xkbBindings {
			binding.object.Enable()
		}
	}

	if focused := s.focused; focused != nil && focused.closed {
		s.focused = nil
	}

	// If no window was interacted with in the current manage sequence,
	// intentionally pass nil to ensure the window on top has focus.
	// This is necessary to handle new windows for example.
	s.focus(s.interacted)
	s.interacted = nil

	if s.pendingAction != nil {
		s.pendingAction()
		s.pendingAction = nil
	}
}
