package config

type ServerConfig struct {
	Root   string
	Thread int
	Daemon string
	Port   int
}

func Default() ServerConfig {
	return ServerConfig{
		Root:   "./",
		Thread: 2,
		Daemon: "",
		Port:   9000,
	}
}
