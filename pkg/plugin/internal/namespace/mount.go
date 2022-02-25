package namespace

// based on https://github.com/teddyking/ns-process/blob/4.1/rootfs.go

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func pivotRoot(newroot string) error {
	putold := filepath.Join(newroot, "/.pivot_root")

	// bind mount newroot to itself - this is a slight hack needed to satisfy the
	// pivot_root requirement that newroot and putold must not be on the same
	// filesystem as the current root
	if err := unix.Mount(newroot, newroot, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return err
	}

	// create putold directory
	if err := os.MkdirAll(putold, 0700); err != nil {
		return err
	}

	// call pivot_root
	if err := unix.PivotRoot(newroot, putold); err != nil {
		return err
	}

	// ensure current working directory is set to new root
	if err := os.Chdir("/"); err != nil {
		return err
	}

	// umount putold, which now lives at /.pivot_root
	putold = "/.pivot_root"
	if err := unix.Unmount(putold, unix.MNT_DETACH); err != nil {
		return err
	}

	// remove putold
	if err := os.RemoveAll(putold); err != nil {
		return err
	}

	return nil
}

func mountProc(newroot string) error {
	target := filepath.Join(newroot, "/proc")

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	return unix.Mount("proc", target, "proc", 0, "")
}

func bindMount(newroot, mount string) error {
	target := filepath.Join(newroot, mount)

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	stat, err := os.Stat(mount)
	if err != nil {
		return err
	}

	// basically a touch call to create the file, otherwise we can't mount to it.
	f, err := os.OpenFile(target, os.O_CREATE, stat.Mode())
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	return unix.Mount(mount, target, "bind", unix.MS_BIND, "")
}
