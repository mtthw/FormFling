package config

import "testing"

func TestParseTrustedProxyCIDRs(t *testing.T) {
	prefixes, err := ParseTrustedProxyCIDRs("192.0.2.1/24, 2001:db8::1/128")
	if err != nil || len(prefixes) != 2 || prefixes[0].String() != "192.0.2.0/24" {
		t.Fatalf("unexpected prefixes %v, error %v", prefixes, err)
	}
	if _, err := ParseTrustedProxyCIDRs("not a CIDR"); err == nil {
		t.Fatal("invalid proxy CIDR should fail")
	}
}
