package config

// TODO: replace this with something cgo-free
// https://codeberg.org/vyivel/croffle/src/branch/main/internal/xkb/xkb.go

// #include <errno.h>
// #include <stdlib.h>
// #include <string.h>
// #include <unistd.h>
//
// #cgo pkg-config: xkbcommon
// #include <xkbcommon/xkbcommon.h>
// #include <xkbcommon/xkbcommon-keysyms.h>
import "C"

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

type KeymapProperty int8

const KeymapFormatTextV2 = C.XKB_KEYMAP_FORMAT_TEXT_V2

const (
	KeymapPropertyRules KeymapProperty = 1 << iota
	KeymapPropertyModel
	KeymapPropertyLayout
	KeymapPropertyVariant
	KeymapPropertyOptions
)

type RMLVO struct {
	Rules      string
	Model      string
	Layout     string
	Variant    string
	Options    string
	Properties KeymapProperty
}

func KeysymFromName(name string) uint32 {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return uint32(C.xkb_keysym_from_name(cName, C.XKB_KEYSYM_CASE_INSENSITIVE))
}

const (
	ModifierNone  uint32 = 0
	ModifierShift uint32 = 1
	ModifierCtrl  uint32 = 4
	ModifierMod1  uint32 = 8
	ModifierMod3  uint32 = 32
	ModifierMod4  uint32 = 64
	ModifierMod5  uint32 = 128
)

func ModifierFromName(name string) (uint32, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "shift":
		return ModifierShift, nil
	case "ctrl", "control":
		return ModifierCtrl, nil
	case "alt", "mod1":
		return ModifierMod1, nil
	case "mod3":
		return ModifierMod3, nil
	case "super", "mod4", "logo", "meta":
		return ModifierMod4, nil
	case "mod5":
		return ModifierMod5, nil
	case "none":
		return ModifierNone, nil
	default:
		return 0, fmt.Errorf("unknown modifier %q", name)
	}
}

func NewKeymapFromPath(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("OS: %w", err)
	}

	ctx := C.xkb_context_new(C.XKB_KEYMAP_COMPILE_NO_FLAGS)
	defer C.xkb_context_unref(ctx)

	xkbKeymap := C.xkb_keymap_new_from_buffer(ctx, (*C.char)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		KeymapFormatTextV2, C.XKB_KEYMAP_COMPILE_NO_FLAGS)
	if xkbKeymap == nil {
		return 0, errors.New("failed to compile a keymap")
	}
	defer C.xkb_keymap_unref(xkbKeymap)

	fd, err := getKeymapFD(xkbKeymap)
	if err != nil {
		return 0, fmt.Errorf("failed to get a keymap FD: %w", err)
	}
	return fd, nil
}

func NewKeymapFromNames(rmlvo *RMLVO) (int, error) {
	var names C.struct_xkb_rule_names
	if rmlvo.Properties&KeymapPropertyRules != 0 {
		names.rules = C.CString(rmlvo.Rules)
		defer C.free(unsafe.Pointer(names.rules))
	}
	if rmlvo.Properties&KeymapPropertyModel != 0 {
		names.model = C.CString(rmlvo.Model)
		defer C.free(unsafe.Pointer(names.model))
	}
	if rmlvo.Properties&KeymapPropertyLayout != 0 {
		names.layout = C.CString(rmlvo.Layout)
		defer C.free(unsafe.Pointer(names.layout))
	}
	if rmlvo.Properties&KeymapPropertyVariant != 0 {
		names.variant = C.CString(rmlvo.Variant)
		defer C.free(unsafe.Pointer(names.variant))
	}
	if rmlvo.Properties&KeymapPropertyOptions != 0 {
		names.options = C.CString(rmlvo.Options)
		defer C.free(unsafe.Pointer(names.options))
	}

	ctx := C.xkb_context_new(C.XKB_KEYMAP_COMPILE_NO_FLAGS)
	defer C.xkb_context_unref(ctx)

	xkbKeymap := C.xkb_keymap_new_from_names2(ctx, &names, KeymapFormatTextV2, C.XKB_KEYMAP_COMPILE_NO_FLAGS)
	if xkbKeymap == nil {
		return 0, errors.New("failed to compile a keymap")
	}
	defer C.xkb_keymap_unref(xkbKeymap)

	fd, err := getKeymapFD(xkbKeymap)
	if err != nil {
		return 0, fmt.Errorf("failed to get a keymap FD: %w", err)
	}
	return fd, nil
}

func getKeymapFD(xkbKeymap *C.struct_xkb_keymap) (int, error) {
	str := C.xkb_keymap_get_as_string(xkbKeymap, KeymapFormatTextV2)
	defer C.free(unsafe.Pointer(str))

	var b [32]byte
	_, _ = rand.Read(b[:])
	path := "/dev/shm/croffleim-keymap- " + hex.EncodeToString(b[:])

	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return 0, fmt.Errorf("unix.Open: %w", err)
	}

	_ = unix.Unlink(path)

	size := C.ssize_t(C.strlen(str) + 1)

	if err := unix.Ftruncate(fd, int64(size)); err != nil {
		_ = unix.Close(fd)
		return 0, fmt.Errorf("unix.Ftruncate: %w", err)
	}

	if _, err := unix.Write(fd, unsafe.Slice((*byte)(unsafe.Pointer(str)), size)); err != nil {
		return 0, fmt.Errorf("write: %w", err)
	}

	return fd, nil
}
