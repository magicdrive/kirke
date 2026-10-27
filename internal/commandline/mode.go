package commandline

import (
	"fmt"
	"strings"
)

const (
	SwitchOn  = "on"
	SwitchOff = "off"
)

type OnOffSwitch string

func (m *OnOffSwitch) Set(value string) error {
	switch v := strings.ToLower(value); v {
	case SwitchOn, SwitchOff:
		*m = OnOffSwitch(v)
		return nil
	default:
		return fmt.Errorf("invalid value: %s. Allowed values are 'on', 'off'", value)
	}
}
func (m *OnOffSwitch) String() string {
	return string(*m)
}
