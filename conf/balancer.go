package conf

import "time"

type BalancerYaml struct {
	MaxLoadBeforeActivate   float64       `yaml:"maxLoadBeforeActivate"`
	MinLoadBeforeDeactivate float64       `yaml:"minLoadBeforeDeactivate"`
	CheckInterval           time.Duration `yaml:"checkInterval"`
	CheckTimeout            time.Duration `yaml:"checkTimeout"`
}

func (by BalancerYaml) GetMaxLoad() float64 {
	if by.MaxLoadBeforeActivate > 1 {
		return 1
	} else if by.MaxLoadBeforeActivate < 0 {
		return 0
	} else if by.MinLoadBeforeDeactivate > by.MaxLoadBeforeActivate {
		return 1
	}
	return by.MaxLoadBeforeActivate
}

func (by BalancerYaml) IsHighLoad(current int, max int) bool {
	if max <= 0 {
		return true
	}
	return float64(current)/float64(max) > by.GetMaxLoad()
}

func (by BalancerYaml) GetMinLoad() float64 {
	if by.MinLoadBeforeDeactivate > 1 {
		return 1
	} else if by.MinLoadBeforeDeactivate < 0 {
		return 0
	} else if by.MaxLoadBeforeActivate < by.MinLoadBeforeDeactivate {
		return 0
	}
	return by.MinLoadBeforeDeactivate
}

func (by BalancerYaml) IsLowLoad(current int, max int) bool {
	if max <= 0 {
		return true
	}
	return float64(current)/float64(max) < by.GetMinLoad()
}

func (by BalancerYaml) GetCheckInterval() time.Duration {
	if by.CheckInterval < 1 {
		return 1 * time.Second
	} else {
		return by.CheckInterval
	}
}

func (by BalancerYaml) GetCheckTimeout() time.Duration {
	if by.CheckTimeout < 0 {
		return 1 * time.Second
	} else {
		return by.CheckTimeout
	}
}
