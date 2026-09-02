package render

import (
	"encoding/json"
	"testing"

	"github.com/openshift/hypershift/api/util/ipnet"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

func TestIPNetJSONRoundtrip(t *testing.T) {
	// Test if ipnet.IPNet survives JSON marshal/unmarshal correctly
	original := hypershiftv1beta1.ClusterNetworking{
		ClusterNetwork: []hypershiftv1beta1.ClusterNetworkEntry{
			{CIDR: *mustParseCIDR("10.128.0.0/14"), HostPrefix: 23},
		},
		ServiceNetwork: []hypershiftv1beta1.ServiceNetworkEntry{
			{CIDR: *mustParseCIDR("172.30.0.0/16")},
		},
		MachineNetwork: []hypershiftv1beta1.MachineNetworkEntry{
			{CIDR: *mustParseCIDR("10.0.0.0/16")},
		},
		NetworkType: hypershiftv1beta1.OVNKubernetes,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	t.Logf("JSON: %s", string(data))

	// Unmarshal back
	var roundtrip hypershiftv1beta1.ClusterNetworking
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify ClusterNetwork
	if len(roundtrip.ClusterNetwork) != 1 {
		t.Fatalf("ClusterNetwork length = %d, want 1", len(roundtrip.ClusterNetwork))
	}
	if roundtrip.ClusterNetwork[0].CIDR.String() != "10.128.0.0/14" {
		t.Errorf("ClusterNetwork CIDR = %v, want 10.128.0.0/14", roundtrip.ClusterNetwork[0].CIDR)
	}
	if roundtrip.ClusterNetwork[0].HostPrefix != 23 {
		t.Errorf("HostPrefix = %v, want 23", roundtrip.ClusterNetwork[0].HostPrefix)
	}

	// Verify ServiceNetwork
	if len(roundtrip.ServiceNetwork) != 1 {
		t.Fatalf("ServiceNetwork length = %d, want 1", len(roundtrip.ServiceNetwork))
	}
	if roundtrip.ServiceNetwork[0].CIDR.String() != "172.30.0.0/16" {
		t.Errorf("ServiceNetwork CIDR = %v, want 172.30.0.0/16", roundtrip.ServiceNetwork[0].CIDR)
	}

	// Verify MachineNetwork
	if len(roundtrip.MachineNetwork) != 1 {
		t.Fatalf("MachineNetwork length = %d, want 1", len(roundtrip.MachineNetwork))
	}
	if roundtrip.MachineNetwork[0].CIDR.String() != "10.0.0.0/16" {
		t.Errorf("MachineNetwork CIDR = %v, want 10.0.0.0/16", roundtrip.MachineNetwork[0].CIDR)
	}

	// Verify NetworkType
	if roundtrip.NetworkType != hypershiftv1beta1.OVNKubernetes {
		t.Errorf("NetworkType = %v, want OVNKubernetes", roundtrip.NetworkType)
	}

	t.Log("✅ ipnet.IPNet JSON roundtrip successful!")
}

func mustParseCIDR(s string) *ipnet.IPNet {
	cidr, err := ipnet.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return cidr
}
