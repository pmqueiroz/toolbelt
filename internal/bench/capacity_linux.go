package bench

import "golang.org/x/sys/unix"

func ReadCapacity(dir string) (Capacity, error) {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return Capacity{}, err
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(dir, &fs); err != nil {
		return Capacity{}, err
	}
	return Capacity{
		Memory: uint64(info.Totalram) * uint64(info.Unit),
		Disk:   fs.Bavail * uint64(fs.Bsize),
	}, nil
}
