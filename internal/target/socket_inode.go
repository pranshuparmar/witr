package target

// socketInodeHasOwner reports whether the inode belongs to a socket we found.
// The map value is the owner UID; root (0) is a valid owner, so presence — not
// a truthy value — is the test.
func socketInodeHasOwner(inodes map[string]int, inode string) bool {
	_, ok := inodes[inode]
	return ok
}
