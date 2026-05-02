package prompt

import (
	"strings"
	"testing"
)

func TestWrapUntrustedContent(t *testing.T) {
	out := WrapUntrustedContent("click here", "abc123def456")
	if !strings.Contains(out, "BEGIN UNTRUSTED WEB CONTENT") {
		t.Fatalf("expected boundary header")
	}
	if !strings.Contains(out, "abc123def456") {
		t.Fatalf("expected canary in wrapped content")
	}
	if !strings.Contains(out, "click here") {
		t.Fatalf("expected original content in wrapped content")
	}
}

func TestQATrustBoundaryRule(t *testing.T) {
	out := QATrustBoundaryRule("deadbeefcafe")
	if !strings.Contains(out, "Trust Boundary") {
		t.Fatalf("expected trust boundary title")
	}
	if !strings.Contains(out, "deadbeefcafe") {
		t.Fatalf("expected canary in rule")
	}
}
