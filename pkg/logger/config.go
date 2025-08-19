package logger

// Config represents logger configuration
type Config struct {
	Level      string `yaml:"level" json:"level"`           // debug, info, warn, error
	Format     string `yaml:"format" json:"format"`         // json, text
	Output     string `yaml:"output" json:"output"`         // stdout, stderr, file path
	File       FileConfig `yaml:"file" json:"file,omitempty"` // file rotation settings
}

// FileConfig represents file logging configuration
type FileConfig struct {
	MaxSize    int  `yaml:"max_size" json:"max_size"`       // MB
	MaxBackups int  `yaml:"max_backups" json:"max_backups"` // number of backup files
	MaxAge     int  `yaml:"max_age" json:"max_age"`         // days
	Compress   bool `yaml:"compress" json:"compress"`       // compress rotated files
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
		File: FileConfig{
			MaxSize:    100, // 100MB
			MaxBackups: 3,
			MaxAge:     30, // 30 days
			Compress:   true,
		},
	}
}
