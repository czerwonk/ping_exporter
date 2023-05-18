package config

import (
	"fmt"
	"time"
)

type duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *duration) UnmarshalYAML(unmashal func(interface{}) error) error {
	var s string
	if err := unmashal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("failed to decode duration: %w", err)
	}
	*d = duration(dur)

	return nil
}

// Duration is a convenience getter.
func (d duration) Duration() time.Duration {
	return time.Duration(d)
}

// Set updates the underlying duration.
func (d *duration) Set(dur time.Duration) {
	*d = duration(dur)
}
