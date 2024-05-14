package cli

import (
	"testing"
)

func TestGetLocalIP(t *testing.T) {
	got := GetLocalIP()

	if len(got) == 0 {
		t.Errorf("GetLocalIP() = %v, want non empty", got)
	}

	//check that no ip is loopback
	for _, ip := range got {
		if ip == "127.0.0.1" || ip == "::1" {
			t.Errorf("GetLocalIP() = %v, want non loopback", got)
		}
	}

	//check that no are the same
	for i, ip := range got {
		for j, ip2 := range got {
			if i != j && ip == ip2 {
				t.Errorf("GetLocalIP() = %v, want unique", got)
			}
		}
	}
}
