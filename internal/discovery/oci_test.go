package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/core"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
)

// ---- helpers ----------------------------------------------------------------

func strPtr(s string) *string { return &s }

func makeInstance(opts ...func(*core.Instance)) core.Instance {
	inst := core.Instance{
		Id:                 strPtr("ocid1.instance.oc1..test"),
		DisplayName:        strPtr("test-instance"),
		Shape:              strPtr("VM.Standard.E4.Flex"),
		AvailabilityDomain: strPtr("AD-1"),
		FaultDomain:        strPtr("FAULT-DOMAIN-1"),
		ImageId:            strPtr("ocid1.image.oc1..img"),
		LifecycleState:     core.InstanceLifecycleStateRunning,
		FreeformTags:       map[string]string{},
		DefinedTags:        map[string]map[string]interface{}{},
	}
	for _, o := range opts {
		o(&inst)
	}
	return inst
}

// ---- hasTag -----------------------------------------------------------------

func TestHasTag_FreeformTagMatch(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{"monitoring": "enabled"}
	})
	if !hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return true for matching freeform tag")
	}
}

func TestHasTag_FreeformTagMismatchValue(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{"monitoring": "disabled"}
	})
	if hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return false when value doesn't match")
	}
}

func TestHasTag_FreeformTagMissingKey(t *testing.T) {
	inst := makeInstance()
	if hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return false when key absent")
	}
}

func TestHasTag_DefinedTagMatch(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.DefinedTags = map[string]map[string]interface{}{
			"ops-namespace": {"monitoring": "enabled"},
		}
	})
	if !hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return true for matching defined tag")
	}
}

func TestHasTag_DefinedTagMismatchValue(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.DefinedTags = map[string]map[string]interface{}{
			"ops-namespace": {"monitoring": "disabled"},
		}
	})
	if hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return false when defined tag value doesn't match")
	}
}

func TestHasTag_DefinedTagNonStringValue(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.DefinedTags = map[string]map[string]interface{}{
			"ops-namespace": {"monitoring": 42}, // non-string value
		}
	})
	if hasTag(inst, "monitoring", "42") {
		t.Error("expected hasTag to return false for non-string defined tag value")
	}
}

func TestHasTag_FreeformTakesPrecedenceOverDefined(t *testing.T) {
	// Freeform matches - defined doesn't - should still return true
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{"monitoring": "enabled"}
		i.DefinedTags = map[string]map[string]interface{}{
			"ns": {"monitoring": "disabled"},
		}
	})
	if !hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return true when freeform matches even if defined doesn't")
	}
}

func TestHasTag_MultipleNamespaces(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.DefinedTags = map[string]map[string]interface{}{
			"ns1": {"other": "x"},
			"ns2": {"monitoring": "enabled"},
		}
	})
	if !hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to find tag in second namespace")
	}
}

func TestHasTag_NilFreeformTags(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = nil
		i.DefinedTags = nil
	})
	// Should not panic
	if hasTag(inst, "monitoring", "enabled") {
		t.Error("expected false with nil tags")
	}
}

// ---- isWindows --------------------------------------------------------------

func TestIsWindows_FreeformTagWindows(t *testing.T) {
	cases := []struct {
		name  string
		osVal string
		want  bool
	}{
		{"lowercase windows", "windows", true},
		{"uppercase WINDOWS", "WINDOWS", true},
		{"mixed case Windows", "Windows", true},
		{"linux", "linux", false},
		{"empty", "", false},
		{"windows-2019", "windows-2019", false}, // only exact match via EqualFold
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst := makeInstance(func(i *core.Instance) {
				if tc.osVal != "" {
					i.FreeformTags = map[string]string{"os": tc.osVal}
				}
			})
			if got := isWindows(inst); got != tc.want {
				t.Errorf("isWindows(%q): got %v, want %v", tc.osVal, got, tc.want)
			}
		})
	}
}

func TestIsWindows_NoOsTag(t *testing.T) {
	inst := makeInstance()
	if isWindows(inst) {
		t.Error("expected isWindows to return false when os tag is absent")
	}
}

// ---- buildLabels ------------------------------------------------------------

func TestBuildLabels_CoreLabelsPresent(t *testing.T) {
	tenancy := config.TenancyConfig{
		Name:      "prod",
		TenancyID: "ocid1.tenancy.oc1..prod",
		Region:    "us-ashburn-1",
	}
	inst := makeInstance(func(i *core.Instance) {
		i.LifecycleState = core.InstanceLifecycleStateRunning
	})
	labels := buildLabels(tenancy, "ocid1.compartment.oc1..comp", inst, "10.0.0.5")

	required := map[string]string{
		"__meta_oci_tenancy_name":   "prod",
		"__meta_oci_tenancy_id":     "ocid1.tenancy.oc1..prod",
		"__meta_oci_region":         "us-ashburn-1",
		"__meta_oci_compartment_id": "ocid1.compartment.oc1..comp",
		"__meta_oci_private_ip":     "10.0.0.5",
		"__meta_oci_instance_state": "RUNNING",
		"__meta_oci_instance_id":    "ocid1.instance.oc1..test",
		"__meta_oci_instance_name":  "test-instance",
		"__meta_oci_display_name":   "test-instance",
		"__meta_oci_shape":          "VM.Standard.E4.Flex",
		"__meta_oci_availability_domain": "AD-1",
		"__meta_oci_fault_domain":   "FAULT-DOMAIN-1",
		"__meta_oci_image_id":       "ocid1.image.oc1..img",
	}

	for key, want := range required {
		if got := labels[key]; got != want {
			t.Errorf("label %q: got %q, want %q", key, got, want)
		}
	}
}

func TestBuildLabels_FreeformTagsExposed(t *testing.T) {
	tenancy := config.TenancyConfig{Name: "t", TenancyID: "tid", Region: "r"}
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{
			"environment": "production",
			"team":        "platform",
		}
	})
	labels := buildLabels(tenancy, "comp", inst, "10.0.0.1")

	if labels["__meta_oci_tag_environment"] != "production" {
		t.Errorf("freeform tag 'environment': got %q, want %q",
			labels["__meta_oci_tag_environment"], "production")
	}
	if labels["__meta_oci_tag_team"] != "platform" {
		t.Errorf("freeform tag 'team': got %q, want %q",
			labels["__meta_oci_tag_team"], "platform")
	}
}

func TestBuildLabels_FreeformTagKeySanitized(t *testing.T) {
	tenancy := config.TenancyConfig{Name: "t", TenancyID: "tid", Region: "r"}
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{
			"My-Tag.Key": "value",
		}
	})
	labels := buildLabels(tenancy, "comp", inst, "10.0.0.1")

	// Key should be lowercased and special chars replaced with _
	if _, ok := labels["__meta_oci_tag_my_tag_key"]; !ok {
		t.Error("expected sanitized tag key '__meta_oci_tag_my_tag_key' not found")
	}
}

func TestBuildLabels_NilOptionalFields(t *testing.T) {
	tenancy := config.TenancyConfig{Name: "t", TenancyID: "tid", Region: "r"}
	inst := core.Instance{
		Id:             nil,
		DisplayName:    nil,
		Shape:          nil,
		AvailabilityDomain: nil,
		FaultDomain:    nil,
		ImageId:        nil,
		LifecycleState: core.InstanceLifecycleStateRunning,
		FreeformTags:   map[string]string{},
		DefinedTags:    map[string]map[string]interface{}{},
	}
	// Should not panic on nil pointer fields
	labels := buildLabels(tenancy, "comp", inst, "10.0.0.1")

	// Mandatory labels must still be present
	if labels["__meta_oci_private_ip"] != "10.0.0.1" {
		t.Error("private IP label missing with nil optional fields")
	}
	// Optional labels should be absent (not panic)
	if _, ok := labels["__meta_oci_instance_id"]; ok {
		t.Error("instance_id should be absent when Id is nil")
	}
}

// ---- sanitizeLabelKey -------------------------------------------------------

func TestSanitizeLabelKey(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"monitoring", "monitoring"},
		{"Monitoring", "monitoring"},
		{"my-tag", "my_tag"},
		{"my.tag.key", "my_tag_key"},
		{"My-Tag.Key", "my_tag_key"},
		{"tag with spaces", "tag_with_spaces"},
		{"tag123", "tag123"},
		{"123tag", "123tag"},
		{"", ""},
		{"__reserved__", "__reserved__"},
		{"CamelCase", "camelcase"},
		{"mixed-UPPER_lower.dot", "mixed_upper_lower_dot"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := sanitizeLabelKey(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeLabelKey(%q): got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---- extractErrorCode -------------------------------------------------------

func TestExtractErrorCode_ContextCanceled(t *testing.T) {
	code := extractErrorCode(context.Canceled)
	if code != "context_canceled" {
		t.Errorf("got %q, want %q", code, "context_canceled")
	}
}

func TestExtractErrorCode_DeadlineExceeded(t *testing.T) {
	code := extractErrorCode(context.DeadlineExceeded)
	if code != "timeout" {
		t.Errorf("got %q, want %q", code, "timeout")
	}
}

func TestExtractErrorCode_URLError(t *testing.T) {
	urlErr := &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("connection refused")}
	code := extractErrorCode(urlErr)
	if code != "network_error" {
		t.Errorf("got %q, want %q", code, "network_error")
	}
}

func TestExtractErrorCode_NetOpError(t *testing.T) {
	netErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}
	code := extractErrorCode(netErr)
	if code != "network_error" {
		t.Errorf("got %q, want %q", code, "network_error")
	}
}

func TestExtractErrorCode_UnknownError(t *testing.T) {
	code := extractErrorCode(errors.New("some unknown error"))
	if code != "unknown" {
		t.Errorf("got %q, want %q", code, "unknown")
	}
}

func TestExtractErrorCode_WrappedContextCanceled(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", context.Canceled)
	code := extractErrorCode(wrapped)
	if code != "context_canceled" {
		t.Errorf("got %q, want %q", code, "context_canceled")
	}
}

func TestExtractErrorCode_WrappedDeadlineExceeded(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", context.DeadlineExceeded)
	code := extractErrorCode(wrapped)
	if code != "timeout" {
		t.Errorf("got %q, want %q", code, "timeout")
	}
}

func TestExtractErrorCode_PlainError_IsUnknown(t *testing.T) {
	err := errors.New("some arbitrary error")
	code := extractErrorCode(err)
	if code != "unknown" {
		t.Errorf("plain error should be 'unknown', got %q", code)
	}
}

func TestExtractErrorCode_WrappedURLError(t *testing.T) {
	inner := &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("connection refused")}
	wrapped := fmt.Errorf("outer: %w", inner)
	code := extractErrorCode(wrapped)
	if code != "network_error" {
		t.Errorf("got %q, want %q", code, "network_error")
	}
}

// ---- discoverTenancy (error path: missing key file) -------------------------

func TestDiscoverTenancy_MissingKeyFile(t *testing.T) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			TagKey:       "monitoring",
			TagValue:     "enabled",
			LinuxPort:    9100,
			WindowsPort:  9182,
			RateLimitRPS: 10,
		},
	}
	tenancy := config.TenancyConfig{
		Name:           "test",
		Region:         "us-ashburn-1",
		TenancyID:      "ocid1.tenancy.oc1..test",
		UserID:         "ocid1.user.oc1..test",
		Fingerprint:    "aa:bb",
		PrivateKeyPath: "/nonexistent/path/to/key.pem",
	}

	_, _, err := discoverTenancy(context.Background(), cfg, tenancy)
	if err == nil {
		t.Error("expected error for missing private key file, got nil")
	}
}

// ---- hasTag edge cases ------------------------------------------------------

func TestHasTag_EmptyKeyAndValue(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{"": ""}
	})
	// Empty key with empty value - should match
	if !hasTag(inst, "", "") {
		t.Error("expected hasTag to match empty key and empty value")
	}
}

func TestHasTag_DefinedTagEmptyNamespace(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.DefinedTags = map[string]map[string]interface{}{
			"ns": {},
		}
	})
	if hasTag(inst, "monitoring", "enabled") {
		t.Error("expected hasTag to return false for empty namespace map")
	}
}

// ---- buildLabels: multiple freeform tags ------------------------------------

func TestBuildLabels_MultipleFreeformTagsSanitized(t *testing.T) {
	tenancy := config.TenancyConfig{Name: "t", TenancyID: "tid", Region: "r"}
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{
			"My.Tag":       "v1",
			"Another-Tag":  "v2",
			"simple":       "v3",
			"123numeric":   "v4",
		}
	})
	labels := buildLabels(tenancy, "comp", inst, "10.0.0.1")

	expected := map[string]string{
		"__meta_oci_tag_my_tag":     "v1",
		"__meta_oci_tag_another_tag": "v2",
		"__meta_oci_tag_simple":     "v3",
		"__meta_oci_tag_123numeric": "v4",
	}
	for key, want := range expected {
		if got := labels[key]; got != want {
			t.Errorf("label %q: got %q, want %q", key, got, want)
		}
	}
}

func TestBuildLabels_EmptyFreeformTags(t *testing.T) {
	tenancy := config.TenancyConfig{Name: "t", TenancyID: "tid", Region: "r"}
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = map[string]string{}
	})
	labels := buildLabels(tenancy, "comp", inst, "10.0.0.1")

	// No __meta_oci_tag_* keys should be present
	for k := range labels {
		if len(k) > len("__meta_oci_tag_") && k[:len("__meta_oci_tag_")] == "__meta_oci_tag_" {
			// Only allow tags that came from non-empty freeform tags
			t.Errorf("unexpected tag label %q for empty freeform tags", k)
			break
		}
	}
}

// ---- isWindows: no freeform tags map ----------------------------------------

func TestIsWindows_NilFreeformTags(t *testing.T) {
	inst := makeInstance(func(i *core.Instance) {
		i.FreeformTags = nil
	})
	// Should not panic
	if isWindows(inst) {
		t.Error("expected false for nil freeform tags")
	}
}
