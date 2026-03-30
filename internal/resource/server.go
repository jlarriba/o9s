package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/metric/v1/metrics"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Server{})
	Alias("srv", "server")
	Alias("servers", "server")
	Alias("instance", "server")
	Alias("vm", "server")
}

type Server struct{}

func (s *Server) Kind() string  { return "server" }
func (s *Server) IDColumn() int { return 1 }

func (s *Server) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "FLAVOR", Width: 12},
		{Name: "IMAGE", Width: 20},
		{Name: "CPU%", Width: 6},
		{Name: "MEM%", Width: 6},
		{Name: "DISK%", Width: 6},
		{Name: "IPs", Width: 0},
	}
}

// serverMetrics holds per-VM metric percentages.
type serverMetrics struct {
	cpuPct  string
	memPct  string
	diskPct string
}

func (s *Server) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	allPages, err := servers.List(computeClient, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing servers: %w", err)
	}
	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, err
	}

	imageNames := buildImageNameMap(ctx, c)
	flavorNames := buildFlavorNameMap(ctx, c)
	metricsMap := fetchServerMetrics(ctx, c)

	rows := make([][]string, 0, len(allServers))
	for _, srv := range allServers {
		m := metricsMap[srv.ID]
		rows = append(rows, []string{
			srv.Name,
			srv.ID,
			srv.Status,
			resolveFlavorName(srv, flavorNames),
			resolveImageName(srv, imageNames),
			m.cpuPct,
			m.memPct,
			m.diskPct,
			extractIPs(srv),
		})
	}
	return rows, nil
}

func (s *Server) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	srv, err := servers.Get(ctx, computeClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting server %s: %w", id, err)
	}

	imageNames := buildImageNameMap(ctx, c)
	flavorNames := buildFlavorNameMap(ctx, c)
	volumeNames := buildVolumeNameMap(ctx, c)

	result := [][2]string{
		{"Name", srv.Name},
		{"ID", srv.ID},
		{"Status", srv.Status},
		{"Progress", fmt.Sprintf("%d", srv.Progress)},
		{"Tenant ID", srv.TenantID},
		{"User ID", srv.UserID},
		{"Host ID", srv.HostID},
		{"Host", srv.Host},
		{"Hostname", derefStr(srv.Hostname)},
		{"Hypervisor Hostname", srv.HypervisorHostname},
		{"Instance Name", srv.InstanceName},
		{"Flavor", resolveFlavorName(*srv, flavorNames)},
		{"Image", resolveImageName(*srv, imageNames)},
		{"Addresses", formatAddresses(*srv)},
		{"Access IPv4", srv.AccessIPv4},
		{"Access IPv6", srv.AccessIPv6},
		{"Security Groups", formatSecurityGroups(srv.SecurityGroups)},
		{"Volumes Attached", formatVolumes(srv.AttachedVolumes, volumeNames)},
		{"Key Name", srv.KeyName},
		{"Config Drive", fmt.Sprintf("%v", srv.ConfigDrive)},
		{"Disk Config", string(srv.DiskConfig)},
		{"Availability Zone", srv.AvailabilityZone},
		{"Launched At", formatTime(srv.LaunchedAt)},
		{"Terminated At", formatTime(srv.TerminatedAt)},
		{"Created", srv.Created.String()},
		{"Updated", srv.Updated.String()},
		{"Power State", formatPowerState(srv.PowerState)},
		{"Task State", srv.TaskState},
		{"VM State", srv.VmState},
		{"Locked", derefBool(srv.Locked)},
		{"Reservation ID", derefStr(srv.ReservationID)},
		{"Launch Index", derefInt(srv.LaunchIndex)},
		{"Root Device Name", derefStr(srv.RootDeviceName)},
		{"Kernel ID", derefStr(srv.KernelID)},
		{"Ramdisk ID", derefStr(srv.RAMDiskID)},
		{"Server Groups", derefStrSlice(srv.ServerGroups)},
		{"Tags", derefStrSlice(srv.Tags)},
	}

	// Fault (only if present)
	if srv.Fault.Message != "" {
		result = append(result, [2]string{"Fault", fmt.Sprintf("%s (code %d)", srv.Fault.Message, srv.Fault.Code)})
	}

	result = append(result, [2]string{"Metadata", fmt.Sprintf("%v", srv.Metadata)})

	return result, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefBool(p *bool) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%v", *p)
}

func derefInt(p *int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

func derefStrSlice(p *[]string) string {
	if p == nil || len(*p) == 0 {
		return ""
	}
	return strings.Join(*p, ", ")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.String()
}

func formatPowerState(ps servers.PowerState) string {
	switch ps {
	case 0:
		return "NOSTATE"
	case 1:
		return "Running"
	case 3:
		return "Paused"
	case 4:
		return "Shutdown"
	case 6:
		return "Crashed"
	case 7:
		return "Suspended"
	default:
		return fmt.Sprintf("%d", ps)
	}
}

func formatSecurityGroups(sgs []map[string]any) string {
	names := make([]string, 0, len(sgs))
	for _, sg := range sgs {
		if name, ok := sg["name"].(string); ok {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func formatVolumes(vols []servers.AttachedVolume, names map[string]string) string {
	parts := make([]string, 0, len(vols))
	for _, v := range vols {
		parts = append(parts, ResolveName(v.ID, names))
	}
	return strings.Join(parts, ", ")
}

func formatAddresses(srv servers.Server) string {
	var parts []string
	for netName, addrs := range srv.Addresses {
		addrList, ok := addrs.([]interface{})
		if !ok {
			continue
		}
		var ips []string
		for _, a := range addrList {
			addrMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			if addr, ok := addrMap["addr"].(string); ok {
				ips = append(ips, addr)
			}
		}
		if len(ips) > 0 {
			parts = append(parts, fmt.Sprintf("%s: %s", netName, strings.Join(ips, ", ")))
		}
	}
	return strings.Join(parts, "; ")
}

// Related returns navigable related resources for a server.
func (s *Server) Related(ctx context.Context, c *client.OpenStack, id string) ([]RelatedResource, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}
	srv, err := servers.Get(ctx, computeClient, id).Extract()
	if err != nil {
		return nil, err
	}

	var related []RelatedResource

	// Attached volumes
	volumeNames := buildVolumeNameMap(ctx, c)
	for _, vol := range srv.AttachedVolumes {
		related = append(related, RelatedResource{
			Kind:        "volume",
			ID:          vol.ID,
			DisplayName: ResolveName(vol.ID, volumeNames),
		})
	}

	// Security groups
	sgMap := buildSecurityGroupIDMap(ctx, c)
	for _, sg := range srv.SecurityGroups {
		name, _ := sg["name"].(string)
		sgID, _ := sg["id"].(string)
		if sgID == "" {
			sgID = sgMap[name]
		}
		if sgID != "" {
			related = append(related, RelatedResource{
				Kind:        "securitygroup",
				ID:          sgID,
				DisplayName: name,
			})
		}
	}

	// Networks
	netNameToID := buildNetworkNameToIDMap(ctx, c)
	for netName := range srv.Addresses {
		if netID, ok := netNameToID[netName]; ok {
			related = append(related, RelatedResource{
				Kind:        "network",
				ID:          netID,
				DisplayName: netName,
			})
		}
	}

	// Floating IPs (from addresses with OS-EXT-IPS:type == "floating")
	fipAddrToID := buildFloatingIPAddrToIDMap(ctx, c)
	for _, addrs := range srv.Addresses {
		addrList, ok := addrs.([]interface{})
		if !ok {
			continue
		}
		for _, a := range addrList {
			addrMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			ipType, _ := addrMap["OS-EXT-IPS:type"].(string)
			addr, _ := addrMap["addr"].(string)
			if ipType == "floating" && addr != "" {
				if fipID, ok := fipAddrToID[addr]; ok {
					related = append(related, RelatedResource{
						Kind:        "floatingip",
						ID:          fipID,
						DisplayName: addr,
					})
				}
			}
		}
	}

	// Image
	if imgID, ok := srv.Image["id"].(string); ok && imgID != "" {
		imageNames := buildImageNameMap(ctx, c)
		related = append(related, RelatedResource{
			Kind:        "image",
			ID:          imgID,
			DisplayName: ResolveName(imgID, imageNames),
		})
	}

	// Flavor
	if flvID, ok := srv.Flavor["id"].(string); ok && flvID != "" {
		flavorNames := buildFlavorNameMap(ctx, c)
		related = append(related, RelatedResource{
			Kind:        "flavor",
			ID:          flvID,
			DisplayName: ResolveName(flvID, flavorNames),
		})
	}

	return related, nil
}

// Metrics returns all ceilometer metrics for a specific VM as key-value pairs.
func (s *Server) Metrics(ctx context.Context, c *client.OpenStack, id string) [][2]string {
	return fetchVMMetrics(ctx, c, id)
}

// ServerPctMetrics holds percentage values for pie chart rendering.
type ServerPctMetrics struct {
	CPU  float64 // 0-100, -1 if unavailable
	Mem  float64
	Disk float64
}

// PctMetrics returns CPU, memory, and disk usage as float64 percentages.
func (s *Server) PctMetrics(ctx context.Context, c *client.OpenStack, id string) ServerPctMetrics {
	m := fetchServerMetrics(ctx, c)
	sm := m[id]
	return ServerPctMetrics{
		CPU:  parsePct(sm.cpuPct),
		Mem:  parsePct(sm.memPct),
		Disk: parsePct(sm.diskPct),
	}
}

func parsePct(s string) float64 {
	s = strings.TrimSuffix(s, "%")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return f
}

func (s *Server) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	computeClient, err := c.Compute()
	if err != nil {
		return err
	}
	return servers.Delete(ctx, computeClient, id).ExtractErr()
}

// fetchServerMetrics queries Aetos for CPU, memory, and disk metrics per VM.
// Returns a map of server ID → serverMetrics with percentage strings.
func fetchServerMetrics(ctx context.Context, c *client.OpenStack) map[string]serverMetrics {
	result := map[string]serverMetrics{}

	metricClient, err := c.Metric()
	if err != nil {
		return result
	}

	// Query all metrics we need in parallel-ish (sequential but fast)
	// Use rate() for ceilometer_cpu since it's a cumulative counter (total ns since boot).
	// rate() returns per-second CPU nanoseconds over the given window.
	cpuRateData := queryMetric(ctx, metricClient, "rate(ceilometer_cpu[5m])")
	vcpusData := queryMetric(ctx, metricClient, "ceilometer_vcpus")
	memUsageData := queryMetric(ctx, metricClient, "ceilometer_memory_usage")
	memTotalData := queryMetric(ctx, metricClient, "ceilometer_memory")
	diskUsageData := queryMetric(ctx, metricClient, "ceilometer_disk_device_usage")
	diskTotalData := queryMetric(ctx, metricClient, "ceilometer_disk_device_capacity")

	// Build per-VM maps: resource label → value
	cpuRateByVM := metricByResource(cpuRateData)
	vcpusByVM := metricByResource(vcpusData)
	memUsageByVM := metricByResource(memUsageData)
	memTotalByVM := metricByResource(memTotalData)
	diskUsageByVM := metricByResourceDisk(diskUsageData)
	diskTotalByVM := metricByResourceDisk(diskTotalData)

	// Collect all VM IDs
	allIDs := map[string]bool{}
	for id := range vcpusByVM {
		allIDs[id] = true
	}
	for id := range memTotalByVM {
		allIDs[id] = true
	}

	for id := range allIDs {
		m := serverMetrics{cpuPct: "-", memPct: "-", diskPct: "-"}

		// CPU%: rate() returns per-second CPU nanoseconds.
		// Divide by 1e9 to get CPU-seconds/s per vCPU, then * 100 for percentage.
		if cpuRate, ok := cpuRateByVM[id]; ok {
			if vcpus, ok := vcpusByVM[id]; ok && vcpus > 0 {
				pct := (cpuRate / (vcpus * 1e9)) * 100
				if pct > 100 {
					pct = 100
				}
				m.cpuPct = fmt.Sprintf("%.0f%%", pct)
			}
		}

		// MEM%: memory_usage / memory * 100
		if usage, ok := memUsageByVM[id]; ok {
			if total, ok := memTotalByVM[id]; ok && total > 0 {
				pct := (usage / total) * 100
				m.memPct = fmt.Sprintf("%.0f%%", pct)
			}
		}

		// DISK%: disk_device_usage (bytes) / disk_device_capacity (bytes) * 100
		if usage, ok := diskUsageByVM[id]; ok {
			if total, ok := diskTotalByVM[id]; ok && total > 0 {
				pct := (usage / total) * 100
				m.diskPct = fmt.Sprintf("%.0f%%", pct)
			}
		}

		result[id] = m
	}

	return result
}

func queryMetric(ctx context.Context, metricClient *gophercloud.ServiceClient, query string) *metrics.QueryData {
	r := metrics.Query(ctx, metricClient, metrics.QueryOpts{Query: query})
	data, err := r.Extract()
	if err != nil {
		return nil
	}
	return data
}

// metricByResource extracts resource label → float64 value from query results.
func metricByResource(data *metrics.QueryData) map[string]float64 {
	m := map[string]float64{}
	if data == nil {
		return m
	}
	for _, mv := range data.Result {
		resID := mv.Metric["resource"]
		if resID == "" {
			continue
		}
		val := extractValue(mv)
		// For disk metrics, resource may have "-vda" suffix; strip it
		m[resID] = val
	}
	return m
}

// metricByResourceDisk is like metricByResource but strips device suffixes
// (e.g. "uuid-vda" → "uuid") from the resource label.
func metricByResourceDisk(data *metrics.QueryData) map[string]float64 {
	m := map[string]float64{}
	if data == nil {
		return m
	}
	for _, mv := range data.Result {
		resID := mv.Metric["resource"]
		if resID == "" {
			continue
		}
		// Strip device suffix like "-vda"
		if idx := strings.LastIndex(resID, "-"); idx > 0 && len(resID)-idx <= 4 {
			resID = resID[:idx]
		}
		val := extractValue(mv)
		// Sum across devices
		m[resID] += val
	}
	return m
}

func extractValue(mv metrics.MetricValue) float64 {
	if len(mv.Value) < 2 {
		return 0
	}
	switch v := mv.Value[1].(type) {
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	case float64:
		return v
	}
	return 0
}

// fetchVMMetrics queries all ceilometer metrics for a specific VM and returns
// them as human-readable key-value pairs with proper units.
func fetchVMMetrics(ctx context.Context, c *client.OpenStack, vmID string) [][2]string {
	metricClient, err := c.Metric()
	if err != nil {
		return nil
	}

	// Query all ceilometer metrics for this VM.
	// resource label formats:
	//   exact UUID: "5075b700-..." (cpu, memory, vcpus, disk_root_size, etc.)
	//   UUID-device: "5075b700-...-vda" (disk device metrics)
	//   instance-NNNNN-UUID-tap...: "instance-00000001-5075b700-...-tapXXX" (network metrics)
	query := fmt.Sprintf(`{resource=~".*%s.*",__name__=~"ceilometer_.*"}`, vmID)
	data := queryMetric(ctx, metricClient, query)
	if data == nil || len(data.Result) == 0 {
		return nil
	}

	// Metric display names and unit formatting
	type metricInfo struct {
		label string
		unit  string
	}
	nameMap := map[string]metricInfo{
		"ceilometer_vcpus":                       {"vCPUs", ""},
		"ceilometer_cpu":                         {"CPU Time", "ns"},
		"ceilometer_memory":                      {"Memory Total", "MiB"},
		"ceilometer_memory_usage":                {"Memory Used", "MiB"},
		"ceilometer_memory_available":            {"Memory Available", "MiB"},
		"ceilometer_memory_resident":             {"Memory Resident", "MiB"},
		"ceilometer_memory_swap_in":              {"Memory Swap In", "MiB"},
		"ceilometer_memory_swap_out":             {"Memory Swap Out", "MiB"},
		"ceilometer_disk_root_size":              {"Disk Root Size", "GiB"},
		"ceilometer_disk_ephemeral_size":         {"Disk Ephemeral Size", "GiB"},
		"ceilometer_disk_device_capacity":        {"Disk Device Capacity", "B"},
		"ceilometer_disk_device_allocation":      {"Disk Device Allocation", "B"},
		"ceilometer_disk_device_usage":           {"Disk Device Usage", "B"},
		"ceilometer_disk_device_read_bytes":      {"Disk Read", "B"},
		"ceilometer_disk_device_write_bytes":     {"Disk Write", "B"},
		"ceilometer_disk_device_read_requests":   {"Disk Read Requests", ""},
		"ceilometer_disk_device_write_requests":  {"Disk Write Requests", ""},
		"ceilometer_disk_device_read_latency":    {"Disk Read Latency", "ns"},
		"ceilometer_disk_device_write_latency":   {"Disk Write Latency", "ns"},
		"ceilometer_network_incoming_bytes":      {"Network RX", "B"},
		"ceilometer_network_outgoing_bytes":      {"Network TX", "B"},
		"ceilometer_network_incoming_packets":    {"Network RX Packets", ""},
		"ceilometer_network_outgoing_packets":    {"Network TX Packets", ""},
		"ceilometer_network_incoming_bytes_delta":  {"Network RX Delta", "B"},
		"ceilometer_network_outgoing_bytes_delta":  {"Network TX Delta", "B"},
		"ceilometer_network_incoming_packets_drop": {"Network RX Drops", ""},
		"ceilometer_network_outgoing_packets_drop": {"Network TX Drops", ""},
		"ceilometer_network_incoming_packets_error": {"Network RX Errors", ""},
		"ceilometer_network_outgoing_packets_error": {"Network TX Errors", ""},
		"ceilometer_power_state":                {"Power State", ""},
		"ceilometer_compute_instance_booting_time": {"Boot Time", "s"},
	}

	var result [][2]string
	for _, mv := range data.Result {
		metricName := mv.Metric["__name__"]
		val := extractValue(mv)

		info, ok := nameMap[metricName]
		if !ok {
			// Unknown metric — show raw name
			info = metricInfo{label: metricName, unit: mv.Metric["unit"]}
		}

		valStr := formatMetricValue(val, info.unit)
		result = append(result, [2]string{info.label, valStr})
	}

	return result
}

func formatMetricValue(val float64, unit string) string {
	switch unit {
	case "B":
		return formatBytesFloat(val)
	case "ns":
		return formatNanoseconds(val)
	case "MiB":
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d MiB", int64(val))
		}
		return fmt.Sprintf("%.1f MiB", val)
	case "GiB":
		return fmt.Sprintf("%d GiB", int64(val))
	case "s":
		return fmt.Sprintf("%.2f s", val)
	default:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	}
}

func formatBytesFloat(b float64) string {
	const gib = 1024 * 1024 * 1024
	const mib = 1024 * 1024
	const kib = 1024
	if b >= gib {
		return fmt.Sprintf("%.1f GiB", b/gib)
	}
	if b >= mib {
		return fmt.Sprintf("%.1f MiB", b/mib)
	}
	if b >= kib {
		return fmt.Sprintf("%.1f KiB", b/kib)
	}
	return fmt.Sprintf("%.0f B", b)
}

func formatNanoseconds(ns float64) string {
	if ns >= 1e9 {
		return fmt.Sprintf("%.2f s", ns/1e9)
	}
	if ns >= 1e6 {
		return fmt.Sprintf("%.2f ms", ns/1e6)
	}
	return fmt.Sprintf("%.0f ns", ns)
}

func buildImageNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	imgClient, err := c.ImageService()
	if err != nil {
		return m
	}
	allPages, err := images.List(imgClient, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		return m
	}
	for _, img := range allImages {
		m[img.ID] = img.Name
	}
	return m
}

func buildFlavorNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	computeClient, err := c.Compute()
	if err != nil {
		return m
	}
	allPages, err := flavors.ListDetail(computeClient, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return m
	}
	for _, f := range allFlavors {
		m[f.ID] = f.Name
	}
	return m
}

func resolveFlavorName(srv servers.Server, names map[string]string) string {
	if name, ok := srv.Flavor["original_name"].(string); ok {
		return name
	}
	if id, ok := srv.Flavor["id"].(string); ok {
		if name, ok := names[id]; ok {
			return name
		}
		return id
	}
	return ""
}

func resolveImageName(srv servers.Server, names map[string]string) string {
	if len(srv.Image) == 0 {
		return "(boot vol)"
	}
	if id, ok := srv.Image["id"].(string); ok {
		if name, ok := names[id]; ok {
			return name
		}
		return id
	}
	return ""
}

func extractIPs(srv servers.Server) string {
	var ips []string
	for _, addrs := range srv.Addresses {
		addrList, ok := addrs.([]interface{})
		if !ok {
			continue
		}
		for _, a := range addrList {
			addrMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			if addr, ok := addrMap["addr"].(string); ok {
				ips = append(ips, addr)
			}
		}
	}
	return strings.Join(ips, ", ")
}
