package conf

import (
	"strings"
	"time"
)

type GCPYaml struct {
	AppScheme            string        `yaml:"appScheme"`
	AppHost              string        `yaml:"host"`
	AppBasePrefix        string        `yaml:"appBasePrefix"`
	ProjectID            string        `yaml:"projectID"`
	Zone                 string        `yaml:"zone"`
	URLMap               string        `yaml:"urlMap"`
	NonAppInstanceGroups []string      `yaml:"nonAppInstanceGroups"`
	ServiceEmail         string        `yaml:"serviceEmail"`
	APITimeout           time.Duration `yaml:"apiTimeout"`
}

func (gy GCPYaml) GetAppScheme() string {
	if gy.AppScheme == "" {
		return "http"
	}
	return gy.AppScheme
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
