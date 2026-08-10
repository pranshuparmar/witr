//go:build darwin

package target

import (
	"reflect"
	"testing"
)

func TestParseUDPBoundEndpoints(t *testing.T) {
	// The local endpoint is 4000 in the first record; the second process only
	// has a connected UDP client whose foreign endpoint is 4000.
	output := "p101\nn127.0.0.1:4000\np202\nn127.0.0.1:52100->127.0.0.1:4000\n"
	got := parseUDPBoundFields(4000, output)
	if want := map[int]struct{}{101: {}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pids = %v, want %v", got, want)
	}
}
