package namespace

// based on https://github.com/teddyking/ns-process/blob/4.1/rootfs.go

import (
	"fmt"
	"io"
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

func copyFile(newroot, path string) error {
	target := filepath.Join(newroot, path)

	// get the source file.. and basically error out if it doesn't exist
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	// basically a touch call to create the file, otherwise we can't mount to it.
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, stat.Mode())
	if err != nil {
		return err
	}
	defer f.Close()

	// copy the source file to our target file in the container
	_, err = io.Copy(f, src)

	return err
}

func writeResolveConf(ips ...string) error {
	if err := os.MkdirAll("/etc", 0755); err != nil {
		return err
	}

	f, err := os.OpenFile("/etc/resolv.conf", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, ip := range ips {
		_, err = fmt.Fprintf(f, "nameserver %s\n", ip)
		if err != nil {
			return err
		}
	}

	return err
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
