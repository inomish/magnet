package windowmanager

import (
	"context"
	"log"
	"os/exec"
	"slices"

	"github.com/inomish/magnet/generated/protocol"
)

type seatOperation interface {
	manage(state *seatOperationState)
	render(state *seatOperationState)
	informStart(window *window)
	informEnd(window *window)
}

type seatOperationState struct {
	window *window
	dx, dy int32
}

type seat struct {
	protocol.RiverSeatV1Stub

	windowManager *WindowManager

	object          protocol.RiverSeatV1
	xkbBindings     []*xkbBinding
	pointerBindings []*pointerBinding

	focused    *window
	hovered    *window
	interacted *window

	pendingAction func()

	operation         seatOperation
	operationState    seatOperationState
	operationReleased bool

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

		leftButton  = 0x110
		rightButton = 0x111
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

	seat.pointerBindings = []*pointerBinding{
		newPointerBinding(seat, leftButton, super, func() {
			if seat.hovered != nil {
				seat.pointerMove(seat.hovered)
			}
		}),
		newPointerBinding(seat, rightButton, super, func() {
			if seat.hovered != nil {
				seat.pointerResize(seat.hovered, protocol.RiverWindowV1EdgesRight|protocol.RiverWindowV1EdgesBottom)
			}
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

func (s *seat) HandleRiverSeatV1OpDelta(ctx context.Context, dx int32, dy int32) {
	s.operationState.dx, s.operationState.dy = dx, dy
}

func (s *seat) HandleRiverSeatV1OpRelease(ctx context.Context) {
	s.operationReleased = true
}

func (s *seat) HandleRiverSeatV1PointerEnter(ctx context.Context, riverWindow protocol.RiverWindowV1) {
	s.hovered = riverWindow.UserData().(*window)
}

func (s *seat) HandleRiverSeatV1PointerLeave(ctx context.Context) {
	s.hovered = nil
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

	for _, binding := range s.pointerBindings {
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

		for _, binding := range s.pointerBindings {
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

	s.handleOperation()

	s.operationReleased = false
}

func (s *seat) handleOperation() {
	if operation := s.operation; operation != nil {
		switch {
		case s.operationReleased:
			operation.informEnd(s.operationState.window)
			fallthrough
		case s.operationState.window.closed:
			s.object.OpEnd()
			s.operation = nil
		default:
			operation.manage(&s.operationState)
		}
	}
}

func (s *seat) render() {
	if operation := s.operation; operation != nil {
		operation.render(&s.operationState)
	}
}

func (s *seat) startOperation(window *window, operation seatOperation) {
	if s.operation != nil {
		return
	}

	s.operation = operation
	s.operationState = seatOperationState{
		window: window,
	}

	s.focus(window)

	s.object.OpStartPointer()
	operation.informStart(window)
}

func (s *seat) pointerMove(window *window) {
	s.startOperation(window, newSeatOperationMove(window))
}

func (s *seat) pointerResize(window *window, edges uint32) {
	s.startOperation(window, newSeatOperationResize(window, edges))
}

type seatOperationMove struct {
	startX, startY int32
}

func newSeatOperationMove(window *window) *seatOperationMove {
	return &seatOperationMove{
		startX: window.x,
		startY: window.y,
	}
}

func (move *seatOperationMove) manage(state *seatOperationState) {}

func (move *seatOperationMove) render(state *seatOperationState) {
	state.window.setPosition(move.startX+state.dx, move.startY+state.dy)
}

func (move *seatOperationMove) informStart(window *window) {}

func (move *seatOperationMove) informEnd(window *window) {}

type seatOperationResize struct {
	startX, startY          int32
	startWidth, startHeight int32
	edges                   uint32
}

func newSeatOperationResize(window *window, edges uint32) *seatOperationResize {
	return &seatOperationResize{
		startX:      window.x,
		startY:      window.y,
		startWidth:  window.width,
		startHeight: window.height,
		edges:       edges,
	}
}

func (resize *seatOperationResize) manage(state *seatOperationState) {
	width, height := resize.startWidth, resize.startHeight
	switch {
	case resize.edges&protocol.RiverWindowV1EdgesLeft != 0:
		width -= state.dx
	case resize.edges&protocol.RiverWindowV1EdgesRight != 0:
		width += state.dx
	}

	switch {
	case resize.edges&protocol.RiverWindowV1EdgesTop != 0:
		height -= state.dy
	case resize.edges&protocol.RiverWindowV1EdgesBottom != 0:
		height += state.dy
	}

	state.window.object.ProposeDimensions(max(width, 1), max(height, 1))
}

func (resize *seatOperationResize) render(state *seatOperationState) {
	x := resize.startX
	if resize.edges&protocol.RiverWindowV1EdgesLeft != 0 {
		x += resize.startWidth - state.window.width
	}

	y := resize.startY
	if resize.edges&protocol.RiverWindowV1EdgesTop != 0 {
		y += resize.startHeight - state.window.height
	}

	state.window.setPosition(x, y)
}

func (resize *seatOperationResize) informStart(window *window) {
	window.object.InformResizeStart()
}

func (resize *seatOperationResize) informEnd(window *window) {
	window.object.InformResizeEnd()
}
