package config

type ServerConfig struct {
	Root                string
	Thread              int
	Daemon              string
	Port                int
	RoomSize            int
	HoleCount           int
	MinPlayersToStart   int
	InitialActiveShrews int
	MaxActiveShrews     int
	InterSpawnMS        int
	MapCycleMS          int
}

func Default() ServerConfig {
	return ServerConfig{
		Root:                "./",
		Thread:              2,
		Daemon:              "",
		Port:                9000,
		RoomSize:            3,
		HoleCount:           9,
		MinPlayersToStart:   3,
		InitialActiveShrews: 1,
		MaxActiveShrews:     1,
		InterSpawnMS:        800,
		MapCycleMS:          16_000,
	}
}
