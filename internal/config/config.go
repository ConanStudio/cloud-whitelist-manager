package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration structure
type Config struct {
	Interval  int       `yaml:"interval"`
	IPSource  IPSource  `yaml:"ip_source"`
	Accounts  []Account `yaml:"accounts"`
	Aliyun    Aliyun    `yaml:"aliyun"` // For backward compatibility
}

// IPSource represents IP source configuration
type IPSource struct {
	Type      string            `yaml:"type"`      // http, command, interface
	URL       string            `yaml:"url"`       // for http type
	Timeout   int               `yaml:"timeout"`   // timeout in seconds
	Headers   map[string]string `yaml:"headers"`   // for http type
	Cmd       string            `yaml:"cmd"`       // for command type
	Interface string            `yaml:"interface"` // for interface type
	IPv6      bool              `yaml:"ipv6"`      // for interface type
}

// Aliyun represents Aliyun configuration
type Aliyun struct {
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	RegionID        string `yaml:"region_id"`
	ECS             ECS    `yaml:"ecs"`
	RDS             RDS    `yaml:"rds"`
	Redis           Redis  `yaml:"redis"`
	CLB             CLB    `yaml:"clb"`
}

// Account represents an Aliyun account configuration
type Account struct {
	Name            string `yaml:"name"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	RegionID        string `yaml:"region_id"`
	ECS             ECS    `yaml:"ecs"`
	RDS             RDS    `yaml:"rds"`
	Redis           Redis  `yaml:"redis"`
	CLB             CLB    `yaml:"clb"`
}

// ECS represents ECS security group configuration
type ECS struct {
	Enabled          bool          `yaml:"enabled"`
	SecurityGroupIDs []SecurityGroup `yaml:"security_groups"`
}

// SecurityGroup represents a single ECS security group configuration
type SecurityGroup struct {
	SecurityGroupID string `yaml:"security_group_id"`
	Port            string `yaml:"port"`  // Support port range like "22", "80/80", "-1/-1", "1/65535"
	Priority        int    `yaml:"priority"`
}

// RDS represents RDS whitelist configuration
type RDS struct {
	Enabled        bool         `yaml:"enabled"`
	InstanceWhitelists []InstanceWhitelist `yaml:"instance_whitelists"`
}

// InstanceWhitelist represents a single RDS instance whitelist configuration
type InstanceWhitelist struct {
	InstanceID    string `yaml:"instance_id"`
	WhitelistName string `yaml:"whitelist_name"`
}

// Redis represents Redis whitelist configuration
type Redis struct {
	Enabled             bool                `yaml:"enabled"`
	InstanceWhitelists  []InstanceWhitelist `yaml:"instance_whitelists"`
}

// CLB represents CLB whitelist configuration
type CLB struct {
	Enabled              bool                `yaml:"enabled"`
	LoadBalancerWhitelists []LoadBalancerWhitelist `yaml:"load_balancer_whitelists"`
}

// LoadBalancerWhitelist represents a single CLB whitelist configuration
type LoadBalancerWhitelist struct {
	AclID string `yaml:"acl_id"`  // ACL ID
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be greater than 0")
	}

	// Validate IP source
	switch c.IPSource.Type {
	case "http":
		if c.IPSource.URL == "" {
			return fmt.Errorf("IP source (http): URL is required")
		}
	case "command":
		if c.IPSource.Cmd == "" {
			return fmt.Errorf("IP source (command): command is required")
		}
	case "interface":
		if c.IPSource.Interface == "" {
			return fmt.Errorf("IP source (interface): interface is required")
		}
	case "":
		return fmt.Errorf("IP source type is required")
	default:
		return fmt.Errorf("unknown IP source type '%s'", c.IPSource.Type)
	}

	// Validate accounts if provided
	if len(c.Accounts) > 0 {
		for i, account := range c.Accounts {
			if account.Name == "" {
				return fmt.Errorf("account %d: name is required", i)
			}
			if account.AccessKeyID == "" {
				return fmt.Errorf("account %d: access_key_id is required", i)
			}
			if account.AccessKeySecret == "" {
				return fmt.Errorf("account %d: access_key_secret is required", i)
			}
			if account.RegionID == "" {
				return fmt.Errorf("account %d: region_id is required", i)
			}

			// Validate ECS configuration if enabled
			if account.ECS.Enabled {
				if len(account.ECS.SecurityGroupIDs) == 0 {
					return fmt.Errorf("account %d: At least one ECS security group must be configured when ECS is enabled", i)
				}
				for j, sg := range account.ECS.SecurityGroupIDs {
					if sg.SecurityGroupID == "" {
						return fmt.Errorf("account %d: ECS security group %d security_group_id is required", i, j)
					}
					if sg.Port == "" {
						return fmt.Errorf("account %d: ECS security group %d port is required", i, j)
					}
					if sg.Priority <= 0 {
						return fmt.Errorf("account %d: ECS security group %d priority must be greater than 0", i, j)
					}
				}
			}

			// Validate RDS configuration if enabled
			if account.RDS.Enabled {
				if len(account.RDS.InstanceWhitelists) == 0 {
					return fmt.Errorf("account %d: At least one RDS instance whitelist must be configured when RDS is enabled", i)
				}
				for j, iw := range account.RDS.InstanceWhitelists {
					if iw.InstanceID == "" {
						return fmt.Errorf("account %d: RDS instance whitelist %d instance_id is required", i, j)
					}
					if iw.WhitelistName == "" {
						return fmt.Errorf("account %d: RDS instance whitelist %d whitelist_name is required", i, j)
					}
				}
			}

			// Validate Redis configuration if enabled
			if account.Redis.Enabled {
				if len(account.Redis.InstanceWhitelists) == 0 {
					return fmt.Errorf("account %d: At least one Redis instance whitelist must be configured when Redis is enabled", i)
				}
				for j, iw := range account.Redis.InstanceWhitelists {
					if iw.InstanceID == "" {
						return fmt.Errorf("account %d: Redis instance whitelist %d instance_id is required", i, j)
					}
					if iw.WhitelistName == "" {
						return fmt.Errorf("account %d: Redis instance whitelist %d whitelist_name is required", i, j)
					}
				}
			}

			// Validate CLB configuration if enabled
			if account.CLB.Enabled {
				if len(account.CLB.LoadBalancerWhitelists) == 0 {
					return fmt.Errorf("account %d: At least one CLB whitelist must be configured when CLB is enabled", i)
				}
				for j, lbw := range account.CLB.LoadBalancerWhitelists {
					if lbw.AclID == "" {
						return fmt.Errorf("account %d: CLB whitelist %d acl_id is required", i, j)
					}
				}
			}
		}
	} else {
		// Validate single Aliyun configuration for backward compatibility
		if c.Aliyun.AccessKeyID == "" {
			return fmt.Errorf("aliyun.access_key_id is required")
		}

		if c.Aliyun.AccessKeySecret == "" {
			return fmt.Errorf("aliyun.access_key_secret is required")
		}

		if c.Aliyun.RegionID == "" {
			return fmt.Errorf("aliyun.region_id is required")
		}

		// Validate ECS configuration if enabled
		if c.Aliyun.ECS.Enabled {
			if len(c.Aliyun.ECS.SecurityGroupIDs) == 0 {
				return fmt.Errorf("aliyun: At least one ECS security group must be configured when ECS is enabled")
			}
			for i, sg := range c.Aliyun.ECS.SecurityGroupIDs {
				if sg.SecurityGroupID == "" {
					return fmt.Errorf("aliyun: ECS security group %d security_group_id is required", i)
				}
				if sg.Port == "" {
					return fmt.Errorf("aliyun: ECS security group %d port is required", i)
				}
				if sg.Priority <= 0 {
					return fmt.Errorf("aliyun: ECS security group %d priority must be greater than 0", i)
				}
			}
		}

		// Validate RDS configuration if enabled
		if c.Aliyun.RDS.Enabled {
			if len(c.Aliyun.RDS.InstanceWhitelists) == 0 {
				return fmt.Errorf("aliyun: At least one RDS instance whitelist must be configured when RDS is enabled")
			}
			for i, iw := range c.Aliyun.RDS.InstanceWhitelists {
				if iw.InstanceID == "" {
					return fmt.Errorf("aliyun: RDS instance whitelist %d instance_id is required", i)
				}
				if iw.WhitelistName == "" {
					return fmt.Errorf("aliyun: RDS instance whitelist %d whitelist_name is required", i)
				}
			}
		}

		// Validate Redis configuration if enabled
		if c.Aliyun.Redis.Enabled {
			if len(c.Aliyun.Redis.InstanceWhitelists) == 0 {
				return fmt.Errorf("aliyun: At least one Redis instance whitelist must be configured when Redis is enabled")
			}
			for i, iw := range c.Aliyun.Redis.InstanceWhitelists {
				if iw.InstanceID == "" {
					return fmt.Errorf("aliyun: Redis instance whitelist %d instance_id is required", i)
				}
				if iw.WhitelistName == "" {
					return fmt.Errorf("aliyun: Redis instance whitelist %d whitelist_name is required", i)
				}
			}
		}

		// Validate CLB configuration if enabled
		if c.Aliyun.CLB.Enabled {
			if len(c.Aliyun.CLB.LoadBalancerWhitelists) == 0 {
				return fmt.Errorf("aliyun: At least one CLB whitelist must be configured when CLB is enabled")
			}
			for i, lbw := range c.Aliyun.CLB.LoadBalancerWhitelists {
				if lbw.AclID == "" {
					return fmt.Errorf("aliyun: CLB whitelist %d acl_id is required", i)
				}
			}
		}
	}

	return nil
}

// GetInterval returns the check interval as time.Duration
func (c *Config) GetInterval() time.Duration {
	return time.Duration(c.Interval) * time.Second
}

// GetAliyun returns the Aliyun configuration for this account
func (a *Account) GetAliyun() *Aliyun {
	return &Aliyun{
		AccessKeyID:     a.AccessKeyID,
		AccessKeySecret: a.AccessKeySecret,
		RegionID:        a.RegionID,
		ECS:             a.ECS,
		RDS:             a.RDS,
		Redis:           a.Redis,
		CLB:             a.CLB,
	}
}