package config

type Config struct {
	DSN     string
	Port    int
	WsPort  int
	Workers int
}

// @Inject
func NewConfig() *Config {
	return &Config{
		DSN:     "postgres://localhost/mydb",
		Port:    8080,
		WsPort:  8081,
		Workers: 4,
	}
}
