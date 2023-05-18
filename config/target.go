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
