package ui

import (
	"errors"

	"github.com/wraient/pair/pkg/config"
)

type Pair struct {
	Label string
	Value string
}

// Define a custom type for menu kinds
type MenuType int

// Define constants for the enum
const (
	List MenuType = iota
	ListWithImage
	UserInput
	UserInputWithDetails
)

// OpenMenu takes a menu type and renders accordingly
func OpenMenu(menuType MenuType, items []Pair) (string, error) {

	conf := config.Get()

	output := ""
	var err error

	if conf.UI.Mode == config.UIModeRofi {
		output, err = ShowRofiMenu(menuType, items)
	} else if conf.UI.Mode == config.UIModeCLI {
		output, err = ShowCLIMenu(menuType, items)
	} else {
		err = errors.New("unknown UI mode")
	}

	return output, err
}
