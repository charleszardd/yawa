package hotkey

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ModifierAlt      uint32 = 0x0001
	ModifierControl  uint32 = 0x0002
	ModifierShift    uint32 = 0x0004
	ModifierWindows  uint32 = 0x0008
	ModifierNoRepeat uint32 = 0x4000
)

type Binding struct {
	Modifiers  uint32
	VirtualKey uint32
}

func Parse(value string) (Binding, error) {
	parts := strings.Split(value, "+")
	if len(parts) < 2 {
		return Binding{}, fmt.Errorf("hotkey %q must include a modifier and key", value)
	}

	binding := Binding{Modifiers: ModifierNoRepeat}
	seen := make(map[string]struct{}, len(parts))
	for _, rawPart := range parts[:len(parts)-1] {
		part := strings.ToLower(strings.TrimSpace(rawPart))
		if _, duplicate := seen[part]; duplicate {
			return Binding{}, fmt.Errorf("hotkey %q contains duplicate component %q", value, rawPart)
		}
		seen[part] = struct{}{}
		switch part {
		case "ctrl", "control":
			binding.Modifiers |= ModifierControl
		case "alt":
			binding.Modifiers |= ModifierAlt
		case "shift":
			binding.Modifiers |= ModifierShift
		case "win", "windows":
			binding.Modifiers |= ModifierWindows
		default:
			return Binding{}, fmt.Errorf("hotkey %q has unsupported modifier %q", value, rawPart)
		}
	}
	virtualKey, err := parseKey(parts[len(parts)-1])
	if err != nil {
		return Binding{}, fmt.Errorf("hotkey %q: %w", value, err)
	}
	binding.VirtualKey = virtualKey
	return binding, nil
}

func parseKey(rawKey string) (uint32, error) {
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if len(key) == 1 {
		character := key[0]
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			return uint32(character), nil
		}
	}
	if strings.HasPrefix(key, "F") {
		functionNumber, err := strconv.Atoi(strings.TrimPrefix(key, "F"))
		if err == nil && functionNumber >= 1 && functionNumber <= 24 {
			return uint32(0x70 + functionNumber - 1), nil
		}
	}
	return 0, fmt.Errorf("unsupported key %q; use A-Z, 0-9, or F1-F24", rawKey)
}

func Canonical(binding Binding) string {
	return fmt.Sprintf("%d:%d", binding.Modifiers&^ModifierNoRepeat, binding.VirtualKey)
}
