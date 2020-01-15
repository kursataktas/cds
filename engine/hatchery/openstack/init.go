package openstack

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	"github.com/ovh/cds/sdk/log"
)

// InitHatchery fetch uri from nova
// then list available models
// then list available images
func (h *HatcheryOpenstack) InitHatchery(ctx context.Context) error {
	workersAlive = map[string]int64{}

	authOpts := gophercloud.AuthOptions{
		Username:         h.Config.User,
		Password:         h.Config.Password,
		AllowReauth:      true,
		IdentityEndpoint: h.Config.Address,
		TenantName:       h.Config.Tenant,
		DomainName:       h.Config.Domain,
	}

	provider, errac := openstack.AuthenticatedClient(authOpts)
	if errac != nil {
		return fmt.Errorf("Unable to openstack.AuthenticatedClient: %v", errac)
	}

	openstackClient, errn := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{Region: h.Config.Region})
	if errn != nil {
		return fmt.Errorf("Unable to openstack.NewComputeV2: %s", errn)
	}
	h.openstackClient = openstackClient

	if err := h.initFlavors(); err != nil {
		log.Warning(ctx, "Error getting flavors: %s", err)
	}

	if err := h.initNetworks(); err != nil {
		log.Warning(ctx, "Error getting networks: %s", err)
	}

	if err := h.initIPStatus(ctx); err != nil {
		log.Warning(ctx, "Error on initIPStatus(): %s", err)
	}

	go h.main(ctx)

	return nil
}

func (h *HatcheryOpenstack) initFlavors() error {
	all, err := flavors.ListDetail(h.openstackClient, nil).AllPages()
	if err != nil {
		return fmt.Errorf("initFlavors> error on flavors.ListDetail: %s", err)
	}
	lflavors, err := flavors.ExtractFlavors(all)
	if err != nil {
		return fmt.Errorf("initFlavors> error on flavors.ExtractFlavors: %s", err)
	}
	h.flavors = lflavors
	return nil
}

func (h *HatcheryOpenstack) initNetworks() error {
	all, err := tenantnetworks.List(h.openstackClient).AllPages()
	if err != nil {
		return fmt.Errorf("initNetworks> Unable to get Network: %s", err)
	}
	nets, err := tenantnetworks.ExtractNetworks(all)
	if err != nil {
		return fmt.Errorf("initNetworks> Unable to get Network: %s", err)
	}
	for _, n := range nets {
		if n.Name == h.Config.NetworkString {
			h.networkID = n.ID
			break
		}
	}
	return nil
}

// initIPStatus initializes ipsInfos to
// add workername on ip belong to openstack-ip-range
// this func is called once, when hatchery is starting
func (h *HatcheryOpenstack) initIPStatus(ctx context.Context) error {
	srvs := h.getServers(ctx)
	log.Info(ctx, "initIPStatus> %d srvs", len(srvs))
	for ip := range ipsInfos.ips {
		log.Info(ctx, "initIPStatus> checking %s", ip)
		for _, s := range srvs {
			if len(s.Addresses) == 0 {
				log.Info(ctx, "initIPStatus> server %s - 0 addr", s.Name)
				continue
			}
			for k, v := range s.Addresses {
				if k != h.Config.NetworkString {
					continue
				}
				switch v.(type) {
				case []interface{}:
					for _, z := range v.([]interface{}) {
						var addr string
						var version int
						for x, y := range z.(map[string]interface{}) {
							if x == "addr" {
								addr = y.(string)
							}
							if x == "version" {
								version = int(y.(float64))
							}
						}
						//we only support IPV4
						if addr != "" && version == 4 {
							log.Debug("initIPStatus> server %s - address %s (checking %s)", s.Name, addr, ip)
							if addr != "" && addr == ip {
								log.Info(ctx, "initIPStatus> worker %s - use IP: %s", s.Name, addr)
								ipsInfos.ips[ip] = ipInfos{workerName: s.Name}
							}
						}
					}
				}
			}
		}
	}
	return nil
}