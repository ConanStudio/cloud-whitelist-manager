package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ConanStudio/cloud-whitelist-manager/internal/aliyun"
	"github.com/ConanStudio/cloud-whitelist-manager/internal/config"
	"github.com/ConanStudio/cloud-whitelist-manager/internal/ip"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	err = cfg.Validate()
	if err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}

	logger.Info("Configuration loaded successfully")

	// Create Aliyun clients for all accounts
	var aliyunClients []*aliyun.Client
	
	// If accounts are configured, use them
	if len(cfg.Accounts) > 0 {
		for _, account := range cfg.Accounts {
			client, err := aliyun.NewClient(account.GetAliyun())
			if err != nil {
				logger.Fatalf("Failed to create Aliyun client for account %s: %v", account.Name, err)
			}
			aliyunClients = append(aliyunClients, client)
			logger.Infof("Aliyun client created for account %s", account.Name)
		}
	} else {
		// Backward compatibility: use single account
		aliyunClient, err := aliyun.NewClient(&cfg.Aliyun)
		if err != nil {
			logger.Fatalf("Failed to create Aliyun client: %v", err)
		}
		aliyunClients = append(aliyunClients, aliyunClient)
		logger.Info("Aliyun client created successfully")
	}

	// State to keep track of current IP
	var currentState struct {
		LastIP    string
		CurrentIP string
	}

	// Create a channel to handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel for the ticker
	ticker := time.NewTicker(cfg.GetInterval())
	defer ticker.Stop()

	// Run the IP update immediately on startup
	logger.Info("Running initial IP update")
	err = updateIP(logger, cfg, aliyunClients, &currentState)
	if err != nil {
		logger.Errorf("Initial IP update failed: %v", err)
	}

	// Main loop
	for {
		select {
		case <-ticker.C:
			logger.Info("Running scheduled IP update")
			err := updateIP(logger, cfg, aliyunClients, &currentState)
			if err != nil {
				logger.Errorf("Scheduled IP update failed: %v", err)
			}
		case <-sigChan:
			logger.Info("Received shutdown signal, exiting...")
			return
		}
	}
}

// updateIP performs the IP update process
func updateIP(logger *logrus.Logger, cfg *config.Config, aliyunClients []*aliyun.Client, state *struct{ LastIP, CurrentIP string }) error {
	// Get current public IP
	currentIP, err := ip.GetPublicIP([]config.IPSource{cfg.IPSource})
	if err != nil {
		return fmt.Errorf("failed to get public IP: %v", err)
	}

	logger.Infof("Current public IP: %s", currentIP)

	// Check if IP has changed
	if currentIP == state.CurrentIP {
		logger.Info("IP has not changed, nothing to update")
		return nil
	}

	// Update state
	state.LastIP = state.CurrentIP
	state.CurrentIP = currentIP

	// Update all accounts
	for i, client := range aliyunClients {
		accountName := "account"
		if len(cfg.Accounts) > 0 && i < len(cfg.Accounts) {
			accountName = cfg.Accounts[i].Name
		}

		logger.Infof("Updating resources for account: %s", accountName)

		// Update ECS security group
		if client.GetConfig().ECS.Enabled {
			logger.Infof("Updating ECS security group for account: %s", accountName)
			err := client.UpdateECSWhitelist(state.LastIP, state.CurrentIP)
			if err != nil {
				logger.Errorf("Failed to update ECS whitelist for account %s: %v. Please check if the security group ID is correct and the AccessKey has proper permissions.", accountName, err)
			} else {
				logger.Infof("ECS security group updated successfully for account: %s", accountName)
			}
		}

		// Update RDS whitelist
		if client.GetConfig().RDS.Enabled {
			logger.Infof("Updating RDS whitelist for account: %s", accountName)
			err := client.UpdateRDSWhitelist(state.LastIP, state.CurrentIP)
			if err != nil {
				logger.Errorf("Failed to update RDS whitelist for account %s: %v. Please check if the RDS instance ID is correct and the AccessKey has proper permissions.", accountName, err)
			} else {
				logger.Infof("RDS whitelist updated successfully for account: %s", accountName)
			}
		}

		// Update Redis whitelist
		if client.GetConfig().Redis.Enabled {
			logger.Infof("Updating Redis whitelist for account: %s", accountName)
			err := client.UpdateRedisWhitelist(state.LastIP, state.CurrentIP)
			if err != nil {
				logger.Errorf("Failed to update Redis whitelist for account %s: %v. Please check if the Redis instance ID is correct and the AccessKey has proper permissions.", accountName, err)
			} else {
				logger.Infof("Redis whitelist updated successfully for account: %s", accountName)
			}
		}

		// Update CLB whitelist
		if client.GetConfig().CLB.Enabled {
			logger.Infof("Updating CLB whitelist for account: %s", accountName)
			err := client.UpdateCLBWhitelist(state.LastIP, state.CurrentIP)
			if err != nil {
				logger.Errorf("Failed to update CLB whitelist for account %s: %v. Please check if the CLB instance ID is correct and the AccessKey has proper permissions.", accountName, err)
			} else {
				logger.Infof("CLB whitelist updated successfully for account: %s", accountName)
			}
		}
	}

	logger.Infof("IP updated from %s to %s", state.LastIP, state.CurrentIP)
	return nil
}