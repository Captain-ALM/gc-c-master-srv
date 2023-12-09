package conf

import "time"

type ListenYaml struct {
	Web           string        `yaml:"web"`
	ReadTimeout   time.Duration `yaml:"readTimeout"`
	WriteTimeout  time.Duration `yaml:"writeTimeout"`
	Domains       []string      `yaml:"domains"`
	BasePrefixURL string        `yaml:"basePrefixURL"`
}

func (ly ListenYaml) GetReadTimeout() time.Duration {
	if ly.ReadTimeout.Seconds() < 1 {
		return 1 * time.Second
	} else {
		return ly.ReadTimeout
	}
}

func (ly ListenYaml) GetWriteTimeout() time.Duration {
	if ly.WriteTimeout.Seconds() < 1 {
		return 1 * time.Second
	} else {
		return ly.WriteTimeout
	}
}
