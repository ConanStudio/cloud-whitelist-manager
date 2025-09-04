package aliyun

import (
	"testing"

	"github.com/ConanStudio/cloud-whitelist-manager/internal/config"
)

func TestECSConfigStructures(t *testing.T) {
	// Test that the ECS configuration structures work as expected
	sg := config.SecurityGroup{
		SecurityGroupID: "sg-12345",
		Port:            "22",
		Priority:        100,
	}

	if sg.SecurityGroupID != "sg-12345" {
		t.Errorf("Expected sg-12345, got %s", sg.SecurityGroupID)
	}

	if sg.Port != "22" {
		t.Errorf("Expected 22, got %s", sg.Port)
	}

	if sg.Priority != 100 {
		t.Errorf("Expected 100, got %d", sg.Priority)
	}
}

func TestRDSConfigStructures(t *testing.T) {
	// Test that the RDS configuration structures work as expected
	iw := config.InstanceWhitelist{
		InstanceID:    "rm-12345",
		WhitelistName: "default",
	}

	if iw.InstanceID != "rm-12345" {
		t.Errorf("Expected rm-12345, got %s", iw.InstanceID)
	}

	if iw.WhitelistName != "default" {
		t.Errorf("Expected default, got %s", iw.WhitelistName)
	}
}

func TestCLBConfigStructures(t *testing.T) {
	// Test that the CLB configuration structures work as expected
	lbw := config.LoadBalancerWhitelist{
		AclID: "acl-12345",
	}

	if lbw.AclID != "acl-12345" {
		t.Errorf("Expected acl-12345, got %s", lbw.AclID)
	}
}