package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// When adding more, check if they are actually supported by golang to begin with
// by inspecting the output of `go tool dist list`
var platforms = []string{"linux"}
var arches = []string{"amd64", "arm64"}

func main() {
	outdir := flag.String("outdir", ".", "The directory to put the output file in")
	debug := flag.Bool("debug", false, "Should debug symbols be included in the binaries or not, creates larger packages")

	flag.Parse()

	if len(flag.Args()) != 1 {
		logrus.Fatalf("Requires 1 argument, got %d", len(flag.Args()))
	}
	path := flag.Arg(0)

	manifest, err := plugin.ParseManifestFromFile(path)
	if err != nil {
		logrus.Fatal(err)
	}

	fout, err := os.OpenFile(filepath.Join(*outdir, fmt.Sprintf("%s.plugin", manifest.Name)), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		logrus.Fatal(err)
	}
	defer fout.Close()

	gw, err := gzip.NewWriterLevel(fout, gzip.BestCompression)
	if err != nil {
		logrus.Fatal(err)
	}
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = writeManifest(tw, manifest)
	if err != nil {
		logrus.Fatal(err)
	}

	err = os.Chdir(path)
	if err != nil {
		logrus.Fatal(err)
	}

	for _, platform := range platforms {
		for _, arch := range arches {
			err = buildPlugin(path, tw, platform, arch, *debug)
			if err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

func writeManifest(tw *tar.Writer, manifest *plugin.Manifest) error {
	var buf bytes.Buffer

	err := yaml.NewEncoder(&buf).Encode(manifest)
	if err != nil {
		return err
	}

	err = tw.WriteHeader(&tar.Header{
		Name: "plugin.manifest.yml",
		Mode: 0400,
		Size: int64(buf.Len()),
	})
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(tw)
	return err
}

func buildPlugin(path string, tw *tar.Writer, goos, goarch string, debug bool) error {
	filename := fmt.Sprintf("plugin-%s-%s", goos, goarch)
	logrus.Infof("Building %s", filename)

	// TODO: Add support for non-golang plugins?
	cmd := exec.Command("go", "build", "-a", "-ldflags", "-extldflags -static")
	if !debug {
		cmd.Args = append(cmd.Args, "-ldflags=-s -w")
	}
	cmd.Args = append(cmd.Args, "-o", filename, "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", goos),
		fmt.Sprintf("GOARCH=%s", goarch),
		"CGO_ENABLED=0",
	)

	err := cmd.Run()
	if err != nil {
		return err
	}

	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}

	err = tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0500,
		Size: fi.Size(),
	})
	if err != nil {
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
}
