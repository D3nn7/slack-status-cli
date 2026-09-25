package tui

import (
	"errors"
	"fmt"
)

var (
	errNoClient = errors.New("no Slack client configured - store a token in the settings")
	errNoEmoji  = errors.New("please select an emoji")
	errNoText   = errors.New("please enter a status text")
)

func errTokenInvalid(err error) error {
	return fmt.Errorf("invalid token: %w", err)
}

func err0(msg string) error {
	return errors.New(msg)
}
