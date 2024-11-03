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
	switch strings.ToLower(value) {
	case "on", "off":
		*m = OnOffSwitch(value)
		return nil
	default:
		return fmt.Errorf("invalid value: %s. Allowed values are 'on', 'off'", value)
	}
}
func (m *OnOffSwitch) String() string {
	return string(*m)
}
