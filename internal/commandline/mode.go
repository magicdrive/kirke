package commandline

import (
	"fmt"
	"strings"
)

type PagerMode string

func (m *PagerMode) Set(value string) error {
	switch strings.ToLower(value) {
	case "auto", "no":
		*m = PagerMode(value)
		return nil
	default:
		return fmt.Errorf("invalid pager mode: %s. Allowed values are 'auto', 'no'", value)
	}
}
func (m *PagerMode) String() string {
	return string(*m)
}

type PointerMode string

func (m *PointerMode) Set(value string) error {
	switch strings.ToLower(value) {
	case "on", "off":
		*m = PointerMode(value)
		return nil
	default:
		return fmt.Errorf("invalid pointer mode: %s. Allowed values are 'on', 'off'", value)
	}
}
func (m *PointerMode) String() string {
	return string(*m)
}
