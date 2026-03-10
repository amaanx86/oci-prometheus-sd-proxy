package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"golang.org/x/time/rate"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
)

// tenancyDiscoverer holds shared state for discovering a single tenancy.
type tenancyDiscoverer struct {
	cfg     *config.Config
	tenancy config.TenancyConfig
	compute core.ComputeClient
	net     core.VirtualNetworkClient
	id      identity.IdentityClient
	limiter *rate.Limiter
	retry   common.RetryPolicy
}

// discoverTenancy returns all matching target groups from a single OCI tenancy.
// Errors from individual compartments are logged and skipped rather than failing
// the entire tenancy, so a partial result is always returned.
func discoverTenancy(ctx context.Context, cfg *config.Config, tenancy config.TenancyConfig) ([]TargetGroup, error) {
	keyContent, err := os.ReadFile(tenancy.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key for tenancy %q: %w", tenancy.Name, err)
	}

	var passphrase *string
	if tenancy.Passphrase != "" {
		passphrase = &tenancy.Passphrase
	}

	provider := common.NewRawConfigurationProvider(
		tenancy.TenancyID,
		tenancy.UserID,
		tenancy.Region,
		tenancy.Fingerprint,
		string(keyContent),
		passphrase,
	)

	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("create compute client for tenancy %q: %w", tenancy.Name, err)
	}

	netClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("create network client for tenancy %q: %w", tenancy.Name, err)
	}

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("create identity client for tenancy %q: %w", tenancy.Name, err)
	}

	// Create rate limiter with burst equal to burst for this tenancy
	rps := cfg.Discovery.RateLimitRPS
	burst := int(rps)
	if burst < 1 {
		burst = 1
	}
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	d := &tenancyDiscoverer{
		cfg:     cfg,
		tenancy: tenancy,
		compute: computeClient,
		net:     netClient,
		id:      identityClient,
		limiter: limiter,
		retry:   common.DefaultRetryPolicy(),
	}

	return d.discover(ctx)
}

// discover is the main entry point for the tenancyDiscoverer, orchestrating
// compartment discovery and target group discovery.
func (d *tenancyDiscoverer) discover(ctx context.Context) ([]TargetGroup, error) {
	// Determine which compartments to scan
	compartmentsToScan := d.tenancy.Compartments

	// If no compartments explicitly configured, auto-discover all compartments in the tenancy
	if len(compartmentsToScan) == 0 {
		discovered, err := d.listAllCompartments(ctx, d.tenancy.TenancyID)
		if err != nil {
			slog.Warn("failed to auto-discover compartments - falling back to root",
				"tenancy", d.tenancy.Name,
				"error", err,
			)
			// Fallback to root compartment (tenancy OCID)
			compartmentsToScan = []string{d.tenancy.TenancyID}
		} else {
			compartmentsToScan = discovered
			slog.Info("auto-discovered compartments",
				"tenancy", d.tenancy.Name,
				"count", len(discovered),
			)
		}
	}

	var groups []TargetGroup
	for _, compartmentID := range compartmentsToScan {
		cGroups, err := d.discoverCompartment(ctx, compartmentID)
		if err != nil {
			slog.Warn("compartment discovery failed - skipping",
				"tenancy", d.tenancy.Name,
				"compartment_id", compartmentID,
				"error", err,
			)
			continue
		}
		groups = append(groups, cGroups...)
	}
	return groups, nil
}

// discoverCompartment lists all running instances in a compartment that match
// the configured tag filter and builds a TargetGroup for each.
func (d *tenancyDiscoverer) discoverCompartment(ctx context.Context, compartmentID string) ([]TargetGroup, error) {
	instances, err := d.listAllInstances(ctx, compartmentID)
	if err != nil {
		return nil, fmt.Errorf("list instances in compartment %q: %w", compartmentID, err)
	}

	var groups []TargetGroup
	for _, instance := range instances {
		if !hasTag(instance, d.cfg.Discovery.TagKey, d.cfg.Discovery.TagValue) {
			continue
		}

		privateIP, err := d.getPrimaryPrivateIP(ctx, compartmentID, *instance.Id)
		if err != nil {
			slog.Warn("could not resolve private IP - skipping instance",
				"tenancy", d.tenancy.Name,
				"instance_id", *instance.Id,
				"error", err,
			)
			continue
		}

		port := d.cfg.Discovery.LinuxPort
		if isWindows(instance) {
			port = d.cfg.Discovery.WindowsPort
		}

		target := fmt.Sprintf("%s:%d", privateIP, port)
		labels := buildLabels(d.tenancy, compartmentID, instance, privateIP)

		groups = append(groups, TargetGroup{
			Targets: []string{target},
			Labels:  labels,
		})
	}
	return groups, nil
}

// listAllInstances pages through all RUNNING instances in a compartment.
func (d *tenancyDiscoverer) listAllInstances(ctx context.Context, compartmentID string) ([]core.Instance, error) {
	var instances []core.Instance
	var page *string

	for {
		if err := d.limiter.Wait(ctx); err != nil {
			return nil, err
		}
		resp, err := d.compute.ListInstances(ctx, core.ListInstancesRequest{
			CompartmentId:  common.String(compartmentID),
			LifecycleState: core.InstanceLifecycleStateRunning,
			Limit:          common.Int(100),
			Page:           page,
			RequestMetadata: common.RequestMetadata{
				RetryPolicy: &d.retry,
			},
		})
		if err != nil {
			return nil, err
		}
		instances = append(instances, resp.Items...)
		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}
	return instances, nil
}

// listAllCompartments recursively lists all child compartments under a parent compartment.
// Starts from the root (tenancy) compartment and discovers the full tree.
func (d *tenancyDiscoverer) listAllCompartments(ctx context.Context, rootCompartmentID string) ([]string, error) {
	var allCompartments []string
	var queue []string

	// Start with root compartment
	queue = append(queue, rootCompartmentID)
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true
		allCompartments = append(allCompartments, current)

		// List child compartments of current compartment
		var page *string
		for {
			if err := d.limiter.Wait(ctx); err != nil {
				return nil, err
			}
			resp, err := d.id.ListCompartments(ctx, identity.ListCompartmentsRequest{
				CompartmentId:          common.String(current),
				AccessLevel:            identity.ListCompartmentsAccessLevelAccessible,
				CompartmentIdInSubtree: common.Bool(false), // direct children only
				Limit:                  common.Int(100),
				Page:                   page,
				RequestMetadata: common.RequestMetadata{
					RetryPolicy: &d.retry,
				},
			})
			if err != nil {
				slog.Warn("failed to list child compartments",
					"parent_compartment_id", current,
					"error", err,
				)
				break // continue with next in queue on error
			}

			for _, comp := range resp.Items {
				if comp.Id != nil && comp.LifecycleState == identity.CompartmentLifecycleStateActive {
					if !visited[*comp.Id] {
						queue = append(queue, *comp.Id)
					}
				}
			}

			if resp.OpcNextPage == nil {
				break
			}
			page = resp.OpcNextPage
		}
	}

	return allCompartments, nil
}

// getPrimaryPrivateIP returns the private IP of the primary VNIC for an instance.
func (d *tenancyDiscoverer) getPrimaryPrivateIP(ctx context.Context, compartmentID, instanceID string) (string, error) {
	var page *string

	for {
		if err := d.limiter.Wait(ctx); err != nil {
			return "", err
		}
		resp, err := d.compute.ListVnicAttachments(ctx, core.ListVnicAttachmentsRequest{
			CompartmentId: common.String(compartmentID),
			InstanceId:    common.String(instanceID),
			Page:          page,
			RequestMetadata: common.RequestMetadata{
				RetryPolicy: &d.retry,
			},
		})
		if err != nil {
			return "", fmt.Errorf("list VNIC attachments: %w", err)
		}

		for _, attachment := range resp.Items {
			if attachment.VnicId == nil {
				continue
			}
			if attachment.LifecycleState != core.VnicAttachmentLifecycleStateAttached {
				continue
			}

			if err := d.limiter.Wait(ctx); err != nil {
				return "", err
			}
			vnicResp, err := d.net.GetVnic(ctx, core.GetVnicRequest{
				VnicId: attachment.VnicId,
				RequestMetadata: common.RequestMetadata{
					RetryPolicy: &d.retry,
				},
			})
			if err != nil {
				slog.Warn("failed to get VNIC details", "vnic_id", *attachment.VnicId, "error", err)
				continue
			}

			vnic := vnicResp.Vnic
			if vnic.IsPrimary != nil && *vnic.IsPrimary && vnic.PrivateIp != nil {
				return *vnic.PrivateIp, nil
			}
		}

		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}
	return "", fmt.Errorf("no primary VNIC with private IP found for instance %s", instanceID)
}

// hasTag checks freeform tags first, then defined tags across all namespaces.
func hasTag(instance core.Instance, key, value string) bool {
	if v, ok := instance.FreeformTags[key]; ok && v == value {
		return true
	}
	for _, nsMap := range instance.DefinedTags {
		if v, ok := nsMap[key]; ok {
			if s, ok := v.(string); ok && s == value {
				return true
			}
		}
	}
	return false
}

// isWindows heuristically detects Windows instances by freeform tag "os".
func isWindows(instance core.Instance) bool {
	if v, ok := instance.FreeformTags["os"]; ok {
		return strings.EqualFold(v, "windows")
	}
	return false
}

// buildLabels constructs the full Prometheus label set for an OCI instance.
// All OCI-specific labels follow the __meta_oci_* convention so they can be
// relabelled or dropped in Prometheus scrape config.
func buildLabels(
	tenancy config.TenancyConfig,
	compartmentID string,
	instance core.Instance,
	privateIP string,
) map[string]string {
	labels := map[string]string{
		"__meta_oci_tenancy_name":   tenancy.Name,
		"__meta_oci_tenancy_id":     tenancy.TenancyID,
		"__meta_oci_region":         tenancy.Region,
		"__meta_oci_compartment_id": compartmentID,
		"__meta_oci_private_ip":     privateIP,
		"__meta_oci_instance_state": string(instance.LifecycleState),
	}

	if instance.Id != nil {
		labels["__meta_oci_instance_id"] = *instance.Id
	}
	if instance.DisplayName != nil {
		labels["__meta_oci_instance_name"] = *instance.DisplayName
		// Expose display name as a top-level job hint for relabelling
		labels["__meta_oci_display_name"] = *instance.DisplayName
	}
	if instance.Shape != nil {
		labels["__meta_oci_shape"] = *instance.Shape
	}
	if instance.AvailabilityDomain != nil {
		labels["__meta_oci_availability_domain"] = *instance.AvailabilityDomain
	}
	if instance.FaultDomain != nil {
		labels["__meta_oci_fault_domain"] = *instance.FaultDomain
	}
	if instance.ImageId != nil {
		labels["__meta_oci_image_id"] = *instance.ImageId
	}

	// Expose all freeform tags as __meta_oci_tag_<key>
	for k, v := range instance.FreeformTags {
		labels["__meta_oci_tag_"+sanitizeLabelKey(k)] = v
	}

	return labels
}

// sanitizeLabelKey converts an arbitrary string to a valid Prometheus label key
// by lowercasing and replacing non-alphanumeric characters with underscores.
func sanitizeLabelKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
