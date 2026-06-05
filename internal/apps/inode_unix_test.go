//go:build unix

package apps

import (
	"os"
	"syscall"
	"testing"
)

// inode returns the inode number of fi, used to detect whether a directory was
// removed and recreated (new inode) versus emptied in place (same inode).
func inode(t *testing.T, fi os.FileInfo) uint64 {
	t.Helper()
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("inode check unsupported on this platform")
	}
	return uint64(st.Ino)
}
