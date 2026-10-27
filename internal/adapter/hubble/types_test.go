package hubble

import (
	"testing"

	"github.com/cilium/cilium/api/v1/flow"
)

func TestIgnoreFlow(t *testing.T) {
	t.Parallel()
	t.Run("ignores nil flow", func(t *testing.T) {
		t.Parallel()
		if !ignoreFlow(nil) {
			t.Error("expected nil flow to be ignored")
		}
	})

	t.Run("ignores flow with nil source", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			Source: nil,
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 80,
					},
				},
			},
		}
		if !ignoreFlow(f) {
			t.Error("expected flow with nil source to be ignored")
		}
	})

	t.Run("ignores flow with empty pod name", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			Source: &flow.Endpoint{
				PodName: "",
			},
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 80,
					},
				},
			},
		}
		if !ignoreFlow(f) {
			t.Error("expected flow with empty pod name to be ignored")
		}
	})

	t.Run("ignores flow with nil L4", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			Source: &flow.Endpoint{
				PodName: "test-pod",
			},
			L4: nil,
		}
		if !ignoreFlow(f) {
			t.Error("expected flow with nil L4 to be ignored")
		}
	})

	t.Run("ignores flow with no TCP or ICMPv4", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			Source: &flow.Endpoint{
				PodName: "test-pod",
			},
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_UDP{
					UDP: &flow.UDP{
						DestinationPort: 53,
					},
				},
			},
		}
		if !ignoreFlow(f) {
			t.Error("expected flow with no TCP or ICMPv4 to be ignored")
		}
	})

	t.Run("does not ignore valid TCP flow", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			Source: &flow.Endpoint{
				PodName: "test-pod",
			},
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 80,
					},
				},
			},
		}
		if ignoreFlow(f) {
			t.Error("expected valid TCP flow to not be ignored")
		}
	})

	t.Run("does not ignore valid ICMPv4 flow", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			Source: &flow.Endpoint{
				PodName: "test-pod",
			},
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_ICMPv4{
					ICMPv4: &flow.ICMPv4{
						Type: 8,
					},
				},
			},
			IP: &flow.IP{
				Destination: "10.0.0.1",
			},
		}
		if ignoreFlow(f) {
			t.Error("expected valid ICMPv4 flow to not be ignored")
		}
	})
}

func TestDestination(t *testing.T) {
	t.Parallel()
	t.Run("returns empty for nil L4", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: nil,
		}
		dest, proto := destination(f)
		if dest != "" || proto != "" {
			t.Errorf("expected empty destination and protocol, got %q and %q", dest, proto)
		}
	})

	t.Run("returns empty for non-TCP non-ICMP flow", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_UDP{
					UDP: &flow.UDP{
						DestinationPort: 53,
					},
				},
			},
		}
		dest, proto := destination(f)
		if dest != "" || proto != "" {
			t.Errorf("expected empty destination and protocol, got %q and %q", dest, proto)
		}
	})

	t.Run("returns destination name for TCP with names", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 443,
					},
				},
			},
			DestinationNames: []string{"example.com"},
		}
		dest, proto := destination(f)
		if dest != "example.com:443" {
			t.Errorf("expected 'example.com:443', got %q", dest)
		}
		if proto != "TCP" {
			t.Errorf("expected 'TCP', got %q", proto)
		}
	})

	t.Run("returns destination pod for TCP with local destination", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 8080,
					},
				},
			},
			Destination: &flow.Endpoint{
				Namespace: "default",
				PodName:   "target-pod",
			},
		}
		dest, proto := destination(f)
		if dest != "default/target-pod:8080" {
			t.Errorf("expected 'default/target-pod:8080', got %q", dest)
		}
		if proto != "TCP" {
			t.Errorf("expected 'TCP', got %q", proto)
		}
	})

	t.Run("returns IP destination for TCP with no names or local dest", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 80,
					},
				},
			},
			IP: &flow.IP{
				Destination: "10.0.0.1",
			},
		}
		dest, proto := destination(f)
		if dest != "10.0.0.1:80" {
			t.Errorf("expected '10.0.0.1:80', got %q", dest)
		}
		if proto != "TCP" {
			t.Errorf("expected 'TCP', got %q", proto)
		}
	})

	t.Run("returns empty for TCP with no destination info", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 80,
					},
				},
			},
		}
		dest, proto := destination(f)
		if dest != "" || proto != "" {
			t.Errorf("expected empty destination and protocol, got %q and %q", dest, proto)
		}
	})

	t.Run("returns IP for ICMPv4 flow", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_ICMPv4{
					ICMPv4: &flow.ICMPv4{
						Type: 8,
					},
				},
			},
			IP: &flow.IP{
				Destination: "10.0.0.1",
			},
		}
		dest, proto := destination(f)
		if dest != "10.0.0.1" {
			t.Errorf("expected '10.0.0.1', got %q", dest)
		}
		if proto != "ping" {
			t.Errorf("expected 'ping', got %q", proto)
		}
	})

	t.Run("destination name takes precedence over local destination", func(t *testing.T) {
		t.Parallel()
		f := &flow.Flow{
			L4: &flow.Layer4{
				Protocol: &flow.Layer4_TCP{
					TCP: &flow.TCP{
						DestinationPort: 443,
					},
				},
			},
			DestinationNames: []string{"example.com"},
			Destination: &flow.Endpoint{
				Namespace: "default",
				PodName:   "target-pod",
			},
		}
		dest, proto := destination(f)
		if dest != "example.com:443" {
			t.Errorf("expected 'example.com:443', got %q", dest)
		}
		if proto != "TCP" {
			t.Errorf("expected 'TCP', got %q", proto)
		}
	})
}
