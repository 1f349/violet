package utils

import (
	"encoding"
	"time"
)

type DurationText time.Duration

var _ encoding.TextMarshaler = DurationText(0)
var _ encoding.TextUnmarshaler = (*DurationText)(nil)

func (d DurationText) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(d).String()), nil
}

func (d *DurationText) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = DurationText(duration)
	return nil
}
