package hotkey

import "testing"

func TestParseSupportedHotkeys(t *testing.T) {
	tests := map[string]Binding{
		"Ctrl+Alt+1":    {Modifiers: ModifierControl | ModifierAlt | ModifierNoRepeat, VirtualKey: '1'},
		"Win+Shift+F12": {Modifiers: ModifierWindows | ModifierShift | ModifierNoRepeat, VirtualKey: 0x7B},
	}
	for input, expected := range tests {
		actual, err := Parse(input)
		if err != nil || actual != expected {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", input, actual, err, expected)
		}
	}
}

func TestParseRejectsInvalidHotkeys(t *testing.T) {
	for _, input := range []string{"1", "Ctrl+Ctrl+A", "Meta+A", "Ctrl+Escape", "Ctrl+F25"} {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) error = nil", input)
		}
	}
}
