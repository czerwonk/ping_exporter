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

func (d TargetConfig) MarshalYAML() (interface{}, error) {
	if d.Labels == nil {
		return d.Addr, nil
	}
	ret := make(map[string]map[string]string)
	ret[d.Addr] = d.Labels
	return ret, nil
}
