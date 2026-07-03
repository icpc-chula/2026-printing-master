package config

import "os"

type Config struct {
	Port        string
	DatabaseDSN string
	StorageDir  string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseDSN: buildDSN(),
		StorageDir:  getEnv("STORAGE_DIR", "./storage"),
	}
}

func buildDSN() string {
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	name := getEnv("DB_NAME", "printingmaster")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return "host=" + host +
		" port=" + port +
		" user=" + user +
		" password=" + password +
		" dbname=" + name +
		" sslmode=" + sslmode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
