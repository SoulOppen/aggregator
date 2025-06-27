package config

type Config struct {
	Config map[string]string
}

func (cfg *Config) SetUser(name string) error {
	if cfg.Config == nil {
		cfg.Config = make(map[string]string)
	}
	cfg.Config["current_user_name"] = name
	write(*cfg)
	return nil

}
