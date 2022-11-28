package main

import (
	"time"

	"golang.org/x/sys/unix"
)

func btime(name string) (time.Time, error) {
	flags := unix.AT_SYMLINK_NOFOLLOW
	mask := unix.STATX_ALL

	var statx unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, name, flags, mask, &statx); err != nil {
		return time.Time{}, err
	}
	// fallback to ctime, and fallback to normal stat in case statx isn't supported?
	return time.Unix(statx.Btime.Sec, int64(statx.Btime.Nsec)), nil
}
