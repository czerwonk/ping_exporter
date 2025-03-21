package config

type TargetConfig struct {
	Addr   string
	Labels map[string]string
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (t *TargetConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// If the input is a string, treat it as the Addr
	var s string
	if err := unmarshal(&s); err == nil {
		t.Addr = s
		t.Labels = nil
		return nil
	}
	// Temporary map to capture raw data
	raw := make(map[string]string)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Extract "host" key into Addr
	if addr, ok := raw["host"]; ok {
		t.Addr = addr
		delete(raw, "host") // Remove from labels
	}

	// Store remaining keys as labels
	t.Labels = raw
	return nil
}

func (t TargetConfig) MarshalYAML() (interface{}, error) {
	// If there are no labels, just return the address as a string
	if len(t.Labels) == 0 {
		return t.Addr, nil
	}

	// Otherwise, construct a map with "host" as Addr and other labels
	m := make(map[string]string)
	m["host"] = t.Addr
	for k, v := range t.Labels {
		m[k] = v
	}

	return m, nil
}
