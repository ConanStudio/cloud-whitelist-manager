package aliyun

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/r-kvstore"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/ConanStudio/cloud-whitelist-manager/internal/config"
)

// Client represents the Aliyun client wrapper
type Client struct {
	ecsClient  *ecs.Client
	rdsClient  *rds.Client
	redisClient *r_kvstore.Client
	clbClient  *slb.Client
	config     *config.Aliyun
}

// NewClient creates a new Aliyun client wrapper
func NewClient(aliyunConfig *config.Aliyun) (*Client, error) {
	// Create credentials
	credential := credentials.NewAccessKeyCredential(aliyunConfig.AccessKeyID, aliyunConfig.AccessKeySecret)

	// Create ECS client
	ecsClient, err := ecs.NewClientWithOptions(aliyunConfig.RegionID, sdk.NewConfig(), credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECS client: %v", err)
	}

	// Create RDS client
	rdsClient, err := rds.NewClientWithOptions(aliyunConfig.RegionID, sdk.NewConfig(), credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %v", err)
	}

	// Create Redis client
	redisClient, err := r_kvstore.NewClientWithOptions(aliyunConfig.RegionID, sdk.NewConfig(), credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %v", err)
	}

	// Create CLB client
	clbClient, err := slb.NewClientWithOptions(aliyunConfig.RegionID, sdk.NewConfig(), credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLB client: %v", err)
	}

	return &Client{
		ecsClient:   ecsClient,
		rdsClient:   rdsClient,
		redisClient: redisClient,
		clbClient:   clbClient,
		config:      aliyunConfig,
	}, nil
}

// UpdateECSWhitelist updates ECS security group whitelist for multiple security groups
func (c *Client) UpdateECSWhitelist(oldIP, newIP string) error {
	if !c.config.ECS.Enabled {
		return nil
	}

	// Remove old IP from all security groups if it exists
	if oldIP != "" {
		for _, sg := range c.config.ECS.SecurityGroupIDs {
			err := c.removeIPFromECSSecurityGroup(oldIP, sg)
			if err != nil {
				return fmt.Errorf("failed to remove old IP from ECS security group %s: %v", sg.SecurityGroupID, err)
			}
		}
	}

	// Add new IP to all security groups
	if newIP != "" {
		for _, sg := range c.config.ECS.SecurityGroupIDs {
			err := c.addIPToECSSecurityGroup(newIP, sg)
			if err != nil {
				return fmt.Errorf("failed to add new IP to ECS security group %s: %v", sg.SecurityGroupID, err)
			}
		}
	}

	return nil
}

// removeIPFromECSSecurityGroup removes an IP from ECS security group
func (c *Client) removeIPFromECSSecurityGroup(ip string, sg config.SecurityGroup) error {
	request := ecs.CreateRevokeSecurityGroupRequest()
	request.Scheme = "https"
	request.SecurityGroupId = sg.SecurityGroupID
	
	// Parse port configuration
	portRange := sg.Port
	ipProtocol := "tcp"
	
	// Handle special port ranges
	if portRange == "-1/-1" {
		ipProtocol = "all"
		portRange = "-1/-1"
	} else if strings.Contains(portRange, "/") {
		// Port range like "80/80" or "1/65535"
		ipProtocol = "tcp"
	} else {
		// Single port like "22", convert to range
		ipProtocol = "tcp"
		portRange = fmt.Sprintf("%s/%s", sg.Port, sg.Port)
	}
	
	request.IpProtocol = ipProtocol
	request.PortRange = portRange
	request.SourceCidrIp = fmt.Sprintf("%s/32", ip)
	request.Priority = fmt.Sprintf("%d", sg.Priority)

	_, err := c.ecsClient.RevokeSecurityGroup(request)
	if err != nil {
		// If the rule doesn't exist, it's not an error for us
		if strings.Contains(err.Error(), "InvalidParam.SourceCidrIp") {
			return nil
		}
		return err
	}

	return nil
}

// addIPToECSSecurityGroup adds an IP to ECS security group
func (c *Client) addIPToECSSecurityGroup(ip string, sg config.SecurityGroup) error {
	request := ecs.CreateAuthorizeSecurityGroupRequest()
	request.Scheme = "https"
	request.SecurityGroupId = sg.SecurityGroupID
	
	// Parse port configuration
	portRange := sg.Port
	ipProtocol := "tcp"
	
	// Handle special port ranges
	if portRange == "-1/-1" {
		ipProtocol = "all"
		portRange = "-1/-1"
	} else if strings.Contains(portRange, "/") {
		// Port range like "80/80" or "1/65535"
		ipProtocol = "tcp"
	} else {
		// Single port like "22", convert to range
		ipProtocol = "tcp"
		portRange = fmt.Sprintf("%s/%s", sg.Port, sg.Port)
	}
	
	request.IpProtocol = ipProtocol
	request.PortRange = portRange
	request.SourceCidrIp = fmt.Sprintf("%s/32", ip)
	request.Priority = fmt.Sprintf("%d", sg.Priority)
	request.Description = "Auto added by cloud-whitelist-manager"

	_, err := c.ecsClient.AuthorizeSecurityGroup(request)
	return err
}

// UpdateRDSWhitelist updates RDS whitelist for multiple instances
func (c *Client) UpdateRDSWhitelist(oldIP, newIP string) error {
	if !c.config.RDS.Enabled {
		return nil
	}

	// Process each RDS instance whitelist
	for _, iw := range c.config.RDS.InstanceWhitelists {
		// Get current whitelist for this instance
		currentWhitelist, err := c.getRDSWhitelist(iw)
		if err != nil {
			return fmt.Errorf("failed to get RDS whitelist for instance %s: %v", iw.InstanceID, err)
		}

		// Update whitelist
		newWhitelist := updateIPList(currentWhitelist, oldIP, newIP)
		
		err = c.setRDSWhitelist(newWhitelist, iw)
		if err != nil {
			return fmt.Errorf("failed to update RDS whitelist for instance %s: %v", iw.InstanceID, err)
		}
	}

	return nil
}

// getRDSWhitelist gets current RDS whitelist for a specific instance
func (c *Client) getRDSWhitelist(iw config.InstanceWhitelist) (string, error) {
	request := rds.CreateDescribeDBInstanceIPArrayListRequest()
	request.Scheme = "https"
	request.DBInstanceId = iw.InstanceID

	response, err := c.rdsClient.DescribeDBInstanceIPArrayList(request)
	if err != nil {
		return "", err
	}

	// Find the whitelist group
	for _, ipArray := range response.Items.DBInstanceIPArray {
		if ipArray.DBInstanceIPArrayName == iw.WhitelistName {
			return ipArray.SecurityIPList, nil
		}
	}

	return "", fmt.Errorf("whitelist group %s not found for RDS instance %s", iw.WhitelistName, iw.InstanceID)
}

// setRDSWhitelist sets RDS whitelist for a specific instance
func (c *Client) setRDSWhitelist(whitelist string, iw config.InstanceWhitelist) error {
	request := rds.CreateModifySecurityIpsRequest()
	request.Scheme = "https"
	request.DBInstanceId = iw.InstanceID
	request.SecurityIps = whitelist
	request.WhitelistNetworkType = "MIX" // Support both VPC and classic
	request.DBInstanceIPArrayName = iw.WhitelistName // Add the whitelist name

	_, err := c.rdsClient.ModifySecurityIps(request)
	return err
}

// GetConfig returns the Aliyun configuration
func (c *Client) GetConfig() *config.Aliyun {
	return c.config
}

// UpdateRedisWhitelist updates Redis whitelist for multiple instances
func (c *Client) UpdateRedisWhitelist(oldIP, newIP string) error {
	if !c.config.Redis.Enabled {
		return nil
	}

	// Process each Redis instance whitelist
	for _, iw := range c.config.Redis.InstanceWhitelists {
		// Get current whitelist for this instance
		currentWhitelist, err := c.getRedisWhitelist(iw)
		if err != nil {
			return fmt.Errorf("failed to get Redis whitelist for instance %s: %v", iw.InstanceID, err)
		}

		// Update whitelist
		newWhitelist := updateIPList(currentWhitelist, oldIP, newIP)
		
		err = c.setRedisWhitelist(newWhitelist, iw)
		if err != nil {
			return fmt.Errorf("failed to update Redis whitelist for instance %s: %v", iw.InstanceID, err)
		}
	}

	return nil
}

// getRedisWhitelist gets current Redis whitelist for a specific instance
func (c *Client) getRedisWhitelist(iw config.InstanceWhitelist) (string, error) {
	request := r_kvstore.CreateDescribeSecurityIpsRequest()
	request.Scheme = "https"
	request.InstanceId = iw.InstanceID

	response, err := c.redisClient.DescribeSecurityIps(request)
	if err != nil {
		return "", err
	}

	// Find the whitelist group
	for _, securityIPGroup := range response.SecurityIpGroups.SecurityIpGroup {
		if securityIPGroup.SecurityIpGroupName == iw.WhitelistName {
			return securityIPGroup.SecurityIpList, nil
		}
	}

	return "", fmt.Errorf("whitelist group %s not found for Redis instance %s", iw.WhitelistName, iw.InstanceID)
}

// setRedisWhitelist sets Redis whitelist for a specific instance
func (c *Client) setRedisWhitelist(whitelist string, iw config.InstanceWhitelist) error {
	request := r_kvstore.CreateModifySecurityIpsRequest()
	request.Scheme = "https"
	request.InstanceId = iw.InstanceID
	request.SecurityIps = whitelist
	request.SecurityIpGroupName = iw.WhitelistName

	_, err := c.redisClient.ModifySecurityIps(request)
	return err
}

// UpdateCLBWhitelist updates CLB whitelist for multiple ACLs
func (c *Client) UpdateCLBWhitelist(oldIP, newIP string) error {
	if !c.config.CLB.Enabled {
		return nil
	}

	// Process each CLB whitelist
	for _, lbw := range c.config.CLB.LoadBalancerWhitelists {
		// Get current whitelist for this ACL
		currentWhitelist, err := c.getCLBWhitelist(lbw)
		if err != nil {
			return fmt.Errorf("failed to get CLB whitelist for ACL %s: %v", lbw.AclID, err)
		}

		// Update whitelist
		newWhitelist := updateIPList(currentWhitelist, oldIP, newIP)
		
		err = c.setCLBWhitelist(newWhitelist, lbw)
		if err != nil {
			return fmt.Errorf("failed to update CLB whitelist for ACL %s: %v", lbw.AclID, err)
		}
	}

	return nil
}

// getCLBWhitelist gets current CLB whitelist for a specific ACL
func (c *Client) getCLBWhitelist(lbw config.LoadBalancerWhitelist) (string, error) {
	request := slb.CreateDescribeAccessControlListAttributeRequest()
	request.Scheme = "https"
	request.AclId = lbw.AclID

	response, err := c.clbClient.DescribeAccessControlListAttribute(request)
	if err != nil {
		return "", err
	}

	// Build whitelist string from entries
	var ips []string
	for _, entry := range response.AclEntrys.AclEntry {
		ips = append(ips, strings.TrimSuffix(entry.AclEntryIP, "/32"))
	}
	return strings.Join(ips, ","), nil
}

// setCLBWhitelist sets CLB whitelist for a specific ACL
func (c *Client) setCLBWhitelist(whitelist string, lbw config.LoadBalancerWhitelist) error {
	// Clear existing entries
	clearRequest := slb.CreateRemoveAccessControlListEntryRequest()
	clearRequest.Scheme = "https"
	clearRequest.AclId = lbw.AclID
	
	// We need to get current entries first to remove them
	entriesRequest := slb.CreateDescribeAccessControlListAttributeRequest()
	entriesRequest.Scheme = "https"
	entriesRequest.AclId = lbw.AclID
	
	entriesResponse, err := c.clbClient.DescribeAccessControlListAttribute(entriesRequest)
	if err != nil {
		return err
	}
	
	if len(entriesResponse.AclEntrys.AclEntry) > 0 {
		// Prepare entries to remove
		var entriesToRemove []map[string]string
		for _, entry := range entriesResponse.AclEntrys.AclEntry {
			entriesToRemove = append(entriesToRemove, map[string]string{
				"entry":   entry.AclEntryIP,
				"comment": entry.AclEntryComment,
			})
		}
		
		// Convert to JSON string
		entriesJSON, _ := json.Marshal(entriesToRemove)
		clearRequest.AclEntrys = string(entriesJSON)
		
		_, err = c.clbClient.RemoveAccessControlListEntry(clearRequest)
		if err != nil {
			return err
		}
	}

	// Add new entries
	if whitelist != "" {
		addRequest := slb.CreateAddAccessControlListEntryRequest()
		addRequest.Scheme = "https"
		addRequest.AclId = lbw.AclID
		
		// Prepare new entries
		ips := strings.Split(whitelist, ",")
		var entriesToAdd []map[string]string
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				entriesToAdd = append(entriesToAdd, map[string]string{
					"entry":   ip + "/32",
					"comment": "Auto added by cloud-whitelist-manager",
				})
			}
		}
		
		if len(entriesToAdd) > 0 {
			entriesJSON, _ := json.Marshal(entriesToAdd)
			addRequest.AclEntrys = string(entriesJSON)
			
			_, err = c.clbClient.AddAccessControlListEntry(addRequest)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// updateIPList updates IP list by removing old IP and adding new IP
func updateIPList(currentList, oldIP, newIP string) string {
	// Split current list into IPs
	ips := strings.Split(currentList, ",")
	ipMap := make(map[string]bool)
	
	// Add all IPs to map
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ipMap[ip] = true
		}
	}
	
	// Remove old IP if it exists
	if oldIP != "" {
		delete(ipMap, oldIP)
	}
	
	// Add new IP if it doesn't exist
	if newIP != "" {
		ipMap[newIP] = true
	}
	
	// Build new list
	var newIPs []string
	for ip := range ipMap {
		newIPs = append(newIPs, ip)
	}
	
	return strings.Join(newIPs, ",")
}