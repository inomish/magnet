package windowmanager

import (
	"context"
	"log"
	"os/exec"
	"slices"
	"strings"

	"github.com/inomish/magnet/generated/protocol"
	"github.com/inomish/magnet/internal/config"
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

func newSeat(
	object protocol.RiverSeatV1,
	windowManager *WindowManager,
	cfg config.Configuration,
) *seat {
	seat := &seat{
		windowManager: windowManager,
		object:        object,
		isNew:         true,
	}

	for _, pair := range cfg.ConfigPairs {
		keysym := pair.Keysym
		mods := pair.Modifiers

		action := pair.Action
		argument := pair.Argument

		switch action {
		case "spawn":
			seat.xkbBindings = append(
				seat.xkbBindings,
				newXkbBinding(seat, keysym, mods, func() {
					spawn(argument)
				}),
			)

		case "do":
			if argument == "close" {
				seat.xkbBindings = append(
					seat.xkbBindings,
					newXkbBinding(seat, keysym, mods, func() {
						if seat.focused != nil {
							seat.focused.object.Close()
						}
					}),
				)
			}

		default:
			log.Printf("unknown action %q\n", action)
		}
	}

	seat.object.SetUserData(seat)

	return seat
}

func spawn(command string) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return
	}

	cmd := exec.Command(fields[0], fields[1:]...)

	if err := cmd.Start(); err != nil {
		log.Printf("something went wrong executing %s: %v", command, err)
		return
	}

	go func() {
		_ = cmd.Wait()
	}()
}

func (s *seat) HandleRiverSeatV1Removed(ctx context.Context) {
	s.removed = true
}

func (s *seat) HandleRiverSeatV1WindowInteraction(ctx context.Context, riverWindow protocol.RiverWindowV1) {
	s.interacted = riverWindow.UserData().(*window)
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
