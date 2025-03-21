package config

type TargetConfig struct {
	Addr   string
	Labels map[string]string
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *TargetConfig) UnmarshalYAML(unmashal func(interface{}) error) error {
	var s string
	if err := unmashal(&s); err == nil {
		d.Addr = s
		return nil
	}

	var x map[string]map[string]string
	if err := unmashal(&x); err != nil {
		return err
	}

	for addr, l := range x {
		d.Addr = addr
		d.Labels = l
	}

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
