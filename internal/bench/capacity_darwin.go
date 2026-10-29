package bench

import "golang.org/x/sys/unix"

func ReadCapacity(dir string) (Capacity, error) {
	memory, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return Capacity{}, err
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(dir, &fs); err != nil {
		return Capacity{}, err
	}
	return Capacity{
		Memory: memory,
		Disk:   fs.Bavail * uint64(fs.Bsize),
	}, nil
}
