package windowmanager

import (
	"context"
	"log"
	"os/exec"
	"slices"

	"github.com/inomish/magnet/generated/protocol"
)

type SeatOperation interface {
	Manage(state *SeatOperationState)
	Render(state *SeatOperationState)
	InformStart(w *Window)
	InformEnd(w *Window)
}

type SeatOperationState struct {
	Window *Window
	Dx, Dy int32
}

type Seat struct {
	protocol.RiverSeatV1Stub

	WindowManager *WindowManager

	Object          protocol.RiverSeatV1
	XkbBindings     []*XkbBinding
	PointerBindings []*PointerBinding

	Focused    *Window
	Hovered    *Window
	Interacted *Window

	PendingAction func()

	Operation         SeatOperation
	OperationState    SeatOperationState
	OperationReleased bool

	New     bool
	Removed bool
}

func NewSeat(object protocol.RiverSeatV1, windowManager *WindowManager) *Seat {
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

	seat := &Seat{
		WindowManager: windowManager,
		Object:        object,
		New:           true,
	}

	seat.XkbBindings = []*XkbBinding{
		NewXkbBinding(seat, spaceSym, super, func() {
			cmd := exec.Command("kitty")

			if err := cmd.Start(); err != nil {
				log.Printf("Failed to launch foot: %v", err)
				return
			}
			go func() { _, _ = cmd.Process.Wait() }()
		}),
		NewXkbBinding(seat, nSym, super, func() {
			if len(seat.WindowManager.Windows) > 0 {
				seat.Focus(seat.WindowManager.Windows[0])
			}
		}),
		NewXkbBinding(seat, qSym, super, func() {
			if seat.Focused != nil {
				seat.Focused.Object.Close()
			}
		}),
		NewXkbBinding(seat, escSym, super, func() {
			seat.WindowManager.WindowManagerV1.ExitSession()
		}),
	}

	seat.PointerBindings = []*PointerBinding{
		NewPointerBinding(seat, leftButton, super, func() {
			if seat.Hovered != nil {
				seat.PointerMove(seat.Hovered)
			}
		}),
		NewPointerBinding(seat, rightButton, super, func() {
			if seat.Hovered != nil {
				seat.PointerResize(seat.Hovered, protocol.RiverWindowV1EdgesRight|protocol.RiverWindowV1EdgesBottom)
			}
		}),
	}

	seat.Object.SetUserData(seat)

	return seat
}

func (s *Seat) Focus(window *Window) {
	windowManager := s.WindowManager

	if window == nil && len(windowManager.Windows) > 0 {
		window = windowManager.Windows[len(windowManager.Windows)-1]
	}

	if s.Focused == window {
		return
	}

	if window != nil {
		s.Object.FocusWindow(window.Object)

		// Raise to top
		idx := slices.Index(windowManager.Windows, window)
		windowManager.Windows = append(windowManager.Windows[:idx], windowManager.Windows[idx+1:]...)
		windowManager.Windows = append(windowManager.Windows, window)

		window.Node.PlaceTop()
	} else {
		s.Object.ClearFocus()
	}

	s.Focused = window
}

func (s *Seat) HandleRiverSeatV1OpDelta(ctx context.Context, dx int32, dy int32) {
	s.OperationState.Dx, s.OperationState.Dy = dx, dy
}

func (s *Seat) HandleRiverSeatV1OpRelease(ctx context.Context) {
	s.OperationReleased = true
}

func (s *Seat) HandleRiverSeatV1PointerEnter(ctx context.Context, window protocol.RiverWindowV1) {
	s.Hovered = window.UserData().(*Window)
}

func (s *Seat) HandleRiverSeatV1PointerLeave(ctx context.Context) {
	s.Hovered = nil
}

func (s *Seat) HandleRiverSeatV1Removed(ctx context.Context) {
	s.Removed = true
}

func (s *Seat) HandleRiverSeatV1WindowInteraction(ctx context.Context, window protocol.RiverWindowV1) {
	s.Interacted = window.UserData().(*Window)
}

func (s *Seat) MaybeDestroy() bool {
	if !s.Removed {
		return false
	}

	for _, binding := range s.XkbBindings {
		binding.Object.Destroy()
	}

	for _, binding := range s.PointerBindings {
		binding.Object.Destroy()
	}

	s.Object.Destroy()

	return true
}

func (s *Seat) Manage() {
	if s.New {
		s.New = false
		for _, binding := range s.XkbBindings {
			binding.Object.Enable()
		}

		for _, binding := range s.PointerBindings {
			binding.Object.Enable()
		}
	}

	if w := s.Focused; w != nil && w.Closed {
		s.Focused = nil
	}

	// If no window was interacted with in the current manage sequence,
	// intentionally pass nil to ensure the window on top has focus.
	// This is necessary to handle new windows for example.
	s.Focus(s.Interacted)
	s.Interacted = nil

	if s.PendingAction != nil {
		s.PendingAction()
		s.PendingAction = nil
	}

	s.HandleOperation()

	s.OperationReleased = false
}

func (s *Seat) HandleOperation() {
	if operation := s.Operation; operation != nil {
		switch {
		case s.OperationReleased:
			operation.InformEnd(s.OperationState.Window)
			fallthrough
		case s.OperationState.Window.Closed:
			s.Object.OpEnd()
			s.Operation = nil
		default:
			operation.Manage(&s.OperationState)
		}
	}
}

func (s *Seat) Render() {
	if op := s.Operation; op != nil {
		op.Render(&s.OperationState)
	}
}

func (s *Seat) StartOperation(window *Window, operation SeatOperation) {
	if s.Operation != nil {
		return
	}

	s.Operation = operation
	s.OperationState = SeatOperationState{
		Window: window,
	}

	s.Focus(window)

	s.Object.OpStartPointer()
	operation.InformStart(window)
}

func (s *Seat) PointerMove(w *Window) {
	s.StartOperation(w, NewSeatOperationMove(w))
}

func (s *Seat) PointerResize(w *Window, edges uint32) {
	s.StartOperation(w, NewSeatOperationResize(w, edges))
}

type SeatOperationMove struct {
	StartX, StartY int32
}

func NewSeatOperationMove(w *Window) *SeatOperationMove {
	return &SeatOperationMove{
		StartX: w.X,
		StartY: w.Y,
	}
}

func (s *SeatOperationMove) Manage(state *SeatOperationState) {}

func (s *SeatOperationMove) Render(state *SeatOperationState) {
	state.Window.SetPosition(s.StartX+state.Dx, s.StartY+state.Dy)
}

func (s *SeatOperationMove) InformStart(w *Window) {}

func (s *SeatOperationMove) InformEnd(w *Window) {}

type SeatOperationResize struct {
	StartX, StartY          int32
	StartWidth, StartHeight int32
	Edges                   uint32
}

func NewSeatOperationResize(w *Window, edges uint32) *SeatOperationResize {
	return &SeatOperationResize{
		StartX:      w.X,
		StartY:      w.Y,
		StartWidth:  w.Width,
		StartHeight: w.Height,
		Edges:       edges,
	}
}

func (s *SeatOperationResize) Manage(state *SeatOperationState) {
	width, height := s.StartWidth, s.StartHeight
	switch {
	case s.Edges&protocol.RiverWindowV1EdgesLeft != 0:
		width -= state.Dx
	case s.Edges&protocol.RiverWindowV1EdgesRight != 0:
		width += state.Dx
	}

	switch {
	case s.Edges&protocol.RiverWindowV1EdgesTop != 0:
		height -= state.Dy
	case s.Edges&protocol.RiverWindowV1EdgesBottom != 0:
		height += state.Dy
	}

	state.Window.Object.ProposeDimensions(max(width, 1), max(height, 1))
}

func (s *SeatOperationResize) Render(state *SeatOperationState) {
	x := s.StartX
	if s.Edges&protocol.RiverWindowV1EdgesLeft != 0 {
		x += s.StartWidth - state.Window.Width
	}

	y := s.StartY
	if s.Edges&protocol.RiverWindowV1EdgesTop != 0 {
		y += s.StartHeight - state.Window.Height
	}

	state.Window.SetPosition(x, y)
}

func (s *SeatOperationResize) InformStart(w *Window) {
	w.Object.InformResizeStart()
}

func (s *SeatOperationResize) InformEnd(w *Window) {
	w.Object.InformResizeEnd()
}
