// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/version"

	"github.com/juju/1.25-upgrade/juju2/juju/names"
	jujuversion "github.com/juju/1.25-upgrade/juju2/version"
)

// Archive writes the executable files found in the given directory in
// gzipped tar format to w.
func Archive(w io.Writer, dir string) error {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	gzw := gzip.NewWriter(w)
	defer closeErrorCheck(&err, gzw)

	tarw := tar.NewWriter(gzw)
	defer closeErrorCheck(&err, tarw)

	for _, ent := range entries {
		h := tarHeader(ent)
		logger.Debugf("adding entry: %#v", h)
		// ignore local umask
		if isExecutable(ent) {
			h.Mode = 0755
		} else {
			h.Mode = 0644
		}
		err := tarw.WriteHeader(h)
		if err != nil {
			return err
		}
		fileName := filepath.Join(dir, ent.Name())
		if err := copyFile(tarw, fileName); err != nil {
			return err
		}
	}
	return nil
}

// archiveAndSHA256 calls Archive with the provided arguments,
// and returns a hex-encoded SHA256 hash of the resulting
// archive.
func archiveAndSHA256(w io.Writer, dir string) (sha256hash string, err error) {
	h := sha256.New()
	if err := Archive(io.MultiWriter(h, w), dir); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), err
}

// copyFile writes the contents of the given file to w.
func copyFile(w io.Writer, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// tarHeader returns a tar file header given the file's stat
// information.
func tarHeader(i os.FileInfo) *tar.Header {
	return &tar.Header{
		Typeflag:   tar.TypeReg,
		Name:       i.Name(),
		Size:       i.Size(),
		Mode:       int64(i.Mode() & 0777),
		ModTime:    i.ModTime(),
		AccessTime: i.ModTime(),
		ChangeTime: i.ModTime(),
		Uname:      "ubuntu",
		Gname:      "ubuntu",
	}
}

// isExecutable returns whether the given info
// represents a regular file executable by (at least) the user.
func isExecutable(i os.FileInfo) bool {
	return i.Mode()&(0100|os.ModeType) == 0100
}

// closeErrorCheck means that we can ensure that
// Close errors do not get lost even when we defer them,
func closeErrorCheck(errp *error, c io.Closer) {
	err := c.Close()
	if *errp == nil {
		*errp = err
	}
}

func findExecutable(execFile string) (string, error) {
	logger.Debugf("looking for: %s", execFile)
	if filepath.IsAbs(execFile) {
		return execFile, nil
	}

	dir, file := filepath.Split(execFile)

	// Now we have two possibilities:
	//   file == path indicating that the PATH was searched
	//   dir != "" indicating that it is a relative path

	if dir == "" {
		path := os.Getenv("PATH")
		for _, name := range filepath.SplitList(path) {
			result := filepath.Join(name, file)
			// Use exec.LookPath() to check if the file exists and is executable`
			f, err := exec.LookPath(result)
			if err == nil {
				return f, nil
			}
		}

		return "", fmt.Errorf("could not find %q in the path", file)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, execFile)), nil
}

func copyExistingJujud(dir string) error {
	// Assume that the user is running juju.
	jujuLocation, err := findExecutable(os.Args[0])
	if err != nil {
		logger.Infof("%v", err)
		return err
	}
	jujudLocation := filepath.Join(filepath.Dir(jujuLocation), names.Jujud)
	logger.Debugf("checking: %s", jujudLocation)
	info, err := os.Stat(jujudLocation)
	if err != nil {
		logger.Infof("couldn't find existing jujud")
		return err
	}
	logger.Infof("Found agent binary to upload (%s)", jujudLocation)
	// TODO(thumper): break this out into a util function.
	// copy the file into the dir.
	source, err := os.Open(jujudLocation)
	if err != nil {
		logger.Infof("open source failed: %v", err)
		return err
	}
	defer source.Close()
	target := filepath.Join(dir, names.Jujud)
	logger.Infof("target: %v", target)
	destination, err := os.OpenFile(target, os.O_RDWR|os.O_TRUNC|os.O_CREATE, info.Mode())
	if err != nil {
		logger.Infof("open destination failed: %v", err)
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}
	return nil
}

func buildJujud(dir string) error {
	logger.Infof("building jujud")
	cmds := [][]string{
		{"go", "build", "-gccgoflags=-static-libgo", "-o", filepath.Join(dir, names.Jujud), "github.com/juju/juju/cmd/jujud"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build command %q failed: %v; %s", args[0], err, out)
		}
	}
	return nil
}

func packageLocalTools(toolsDir string, buildAgent bool) error {
	if !buildAgent {
		if err := copyExistingJujud(toolsDir); err != nil {
			return errors.New("no prepackaged agent available and no jujud binary can be found")
		}
		return nil
	}
	logger.Infof("Building agent binary to upload (%s)", jujuversion.Current.String())
	if err := buildJujud(toolsDir); err != nil {
		return errors.Annotate(err, "cannot build jujud agent binary from source")
	}
	return nil
}

// BundleToolsFunc is a function which can bundle all the current juju tools
// in gzipped tar format to the given writer.
type BundleToolsFunc func(build bool, w io.Writer, forceVersion *version.Number) (version.Binary, string, error)

// Override for testing.
var BundleTools BundleToolsFunc = bundleTools

// bundleTools bundles all the current juju tools in gzipped tar
// format to the given writer.
// If forceVersion is not nil, a FORCE-VERSION file is included in
// the tools bundle so it will lie about its current version number.
func bundleTools(build bool, w io.Writer, forceVersion *version.Number) (tvers version.Binary, sha256Hash string, err error) {
	dir, err := ioutil.TempDir("", "juju-tools")
	if err != nil {
		return version.Binary{}, "", err
	}
	defer os.RemoveAll(dir)
	if err := packageLocalTools(dir, build); err != nil {
		return version.Binary{}, "", err
	}

	// Extract the version number that the jujud binary was built with.
	// This is used to check compatibility with the version of the client
	// being used to bootstrap.
	tvers, err = getVersionFromJujud(dir)
	if err != nil {
		return version.Binary{}, "", errors.Trace(err)
	}

	if forceVersion != nil {
		logger.Debugf("forcing version to %s", forceVersion)
		if err := ioutil.WriteFile(filepath.Join(dir, "FORCE-VERSION"), []byte(forceVersion.String()), 0666); err != nil {
			return version.Binary{}, "", err
		}
	}

	sha256hash, err := archiveAndSHA256(w, dir)
	if err != nil {
		return version.Binary{}, "", err
	}
	return tvers, sha256hash, err
}

var execCommand = exec.Command

func getVersionFromJujud(dir string) (version.Binary, error) {
	path := filepath.Join(dir, names.Jujud)
	cmd := execCommand(path, "version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return version.Binary{}, errors.Errorf("cannot get version from %q: %v; %s", path, err, stderr.String()+stdout.String())
	}
	tvs := strings.TrimSpace(stdout.String())
	tvers, err := version.ParseBinary(tvs)
	if err != nil {
		return version.Binary{}, errors.Errorf("invalid version %q printed by jujud", tvs)
	}
	return tvers, nil
}
