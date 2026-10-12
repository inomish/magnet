package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/inomish/magnet/generated/protocol"
	windowmanager "github.com/inomish/magnet/internal/window_manager"
	"hazelnut.eclair.cafe/wlcl"
)

func run(ctx context.Context) (err error) {
	conn, err := wlcl.Connect(ctx, "")
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Don't pass WAYLAND_DEBUG on to children, the added noise makes
	// debugging the window manager itself impractical.
	// It only matters if it's set when the display is created.
	_ = os.Unsetenv("WAYLAND_DEBUG")

	display := protocol.CreateDisplay(conn)

	windowManager := &windowmanager.WindowManager{}

	windowManager.Registry = display.GetRegistry()
	windowManager.Registry.SetUserData(windowManager)

	if err := wlcl.Roundtrip(ctx, display); err != nil {
		return fmt.Errorf("roundtrip: %w", err)
	}

	if !windowManager.WindowManagerV1.IsSet() || !windowManager.XkbBindingsV1.IsSet() {
		return fmt.Errorf("no required globals")
	}

	windowManager.WindowManagerV1.SetUserData(windowManager)

	for !windowManager.Done {
		if err := conn.Dispatch(ctx); err != nil {
			return fmt.Errorf("dispatch: %w", err)
		}
	}

	return windowManager.Err
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}
