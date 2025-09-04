package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	content := `
interval: 300
ip_source:
  type: http
  url: "http://ipinfo.io/ip"
  timeout: 10
accounts:
- name: "test_account"
  access_key_id: "test_key"
  access_key_secret: "test_secret"
  region_id: "cn-hangzhou"
  ecs:
    enabled: true
    security_groups:
      - security_group_id: "sg-test"
        port: "22"
        priority: 100
  rds:
    enabled: true
    instance_whitelists:
      - instance_id: "rm-test"
        whitelist_name: "default"
  redis:
    enabled: true
    instance_whitelists:
      - instance_id: "r-test"
        whitelist_name: "default"
  clb:
    enabled: true
    load_balancer_whitelists:
      - acl_id: "acl-test"
`
	tmpfile, err := ioutil.TempFile("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading config
	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test config values
	if cfg.Interval != 300 {
		t.Errorf("Expected interval 300, got %d", cfg.Interval)
	}

	if cfg.IPSource.Type != "http" {
		t.Errorf("Expected IP source type 'http', got '%s'", cfg.IPSource.Type)
	}

	if len(cfg.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(cfg.Accounts))
	}

	if cfg.Accounts[0].AccessKeyID != "test_key" {
		t.Errorf("Expected access_key_id 'test_key', got '%s'", cfg.Accounts[0].AccessKeyID)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := &Config{
		Interval: 300,
		IPSource: IPSource{
			Type: "http", URL: "http://ipinfo.io/ip", Timeout: 10,
		},
		Accounts: []Account{
			{
				Name:            "test_account",
				AccessKeyID:     "test_key",
				AccessKeySecret: "test_secret",
				RegionID:        "cn-hangzhou",
				ECS: ECS{
					Enabled: true,
					SecurityGroupIDs: []SecurityGroup{
						{
							SecurityGroupID: "sg-test",
							Port:            "22",
							Priority:        100,
						},
					},
				},
				RDS: RDS{
					Enabled: true,
					InstanceWhitelists: []InstanceWhitelist{
						{
							InstanceID:    "rm-test",
							WhitelistName: "default",
						},
					},
				},
				Redis: Redis{
					Enabled: true,
					InstanceWhitelists: []InstanceWhitelist{
						{
							InstanceID:    "r-test",
							WhitelistName: "default",
						},
					},
				},
				CLB: CLB{
					Enabled: true,
					LoadBalancerWhitelists: []LoadBalancerWhitelist{
						{
							AclID: "acl-test",
						},
					},
				},
			},
		},
	}

	err := validConfig.Validate()
	if err != nil {
		t.Errorf("Valid config should not return error, got: %v", err)
	}

	// Test invalid interval
	invalidConfig := *validConfig
	invalidConfig.Interval = 0
	err = invalidConfig.Validate()
	if err == nil {
		t.Error("Invalid interval should return error")
	}

	// Test missing IP source
	invalidConfig = *validConfig
	invalidConfig.IPSource = IPSource{}
	err = invalidConfig.Validate()
	if err == nil {
		t.Error("Missing IP source should return error")
	}

	// Test invalid HTTP source
	invalidConfig = *validConfig
	invalidConfig.IPSource = IPSource{Type: "http", Timeout: 10}
	err = invalidConfig.Validate()
	if err == nil {
		t.Error("Invalid HTTP source should return error")
	}
}