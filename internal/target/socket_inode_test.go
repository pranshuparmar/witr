package target

import "testing"

// inodes maps socket inode to owner UID. UID 0 (root) is a valid owner, so
// presence in the map — not a truthy UID — decides whether an fd is a match.
func TestSocketInodeHasOwner(t *testing.T) {
	cases := []struct {
		name   string
		inodes map[string]int
		inode  string
		want   bool
	}{
		{"root owned", map[string]int{"123": 0}, "123", true},
		{"non-root owned", map[string]int{"123": 1000}, "123", true},
		{"unknown", map[string]int{"123": 0}, "456", false},
		{"empty", map[string]int{}, "123", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := socketInodeHasOwner(tc.inodes, tc.inode); got != tc.want {
				t.Errorf("socketInodeHasOwner(%v, %q) = %v, want %v", tc.inodes, tc.inode, got, tc.want)
			}
		})
	}
}
