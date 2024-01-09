package conf

import (
	"strings"
	"time"
)

type GCPYaml struct {
	AppRestScheme        string        `yaml:"appRestScheme"`
	AppWSScheme          string        `yaml:"appWSScheme"`
	AppHost              string        `yaml:"host"`
	AppBasePrefix        string        `yaml:"appBasePrefix"`
	ProjectID            string        `yaml:"projectID"`
	Zone                 string        `yaml:"zone"`
	URLMap               string        `yaml:"urlMap"`
	NonAppInstanceGroups []string      `yaml:"nonAppInstanceGroups"`
	ServiceEmail         string        `yaml:"serviceEmail"`
	APITimeout           time.Duration `yaml:"apiTimeout"`
	APIActionTimeout     time.Duration `yaml:"apiActionTimeout"`
	APIActionCooldown    time.Duration `yaml:"apiActionCooldown"`
}

func (gy GCPYaml) GetAppRestScheme() string {
	if gy.AppRestScheme == "" {
		return "http"
	}
	return gy.AppRestScheme
}

func (gy GCPYaml) GetAppWSScheme() string {
	if gy.AppWSScheme == "" {
		return "ws"
	}
	return gy.AppWSScheme
}

func (gy GCPYaml) GetAppHost(cnf ConfigYaml) string {
	if gy.AppHost == "" && len(cnf.Listen.Domains) > 0 {
		return cnf.Listen.Domains[0]
	}
	return gy.AppHost
}

func (gy GCPYaml) GetAppBasePrefix() string {
	if strings.HasPrefix(gy.AppBasePrefix, "/") {
		return gy.AppBasePrefix
	}
	return "/" + gy.AppBasePrefix
}

func (gy GCPYaml) GetAPITimeout() time.Duration {
	if gy.APITimeout < 1 {
		return 1 * time.Second
	} else {
		return gy.APITimeout
	}
}

func (gy GCPYaml) GetAPIActionTimeout() time.Duration {
	if gy.APIActionTimeout < 1 {
		return 1 * time.Second
	} else {
		return gy.APIActionTimeout
	}
}

func (gy GCPYaml) GetAPIActionCooldown() time.Duration {
	if gy.APIActionCooldown < 1 {
		return 50 * time.Millisecond
	} else {
		return gy.APIActionCooldown
	}
}
