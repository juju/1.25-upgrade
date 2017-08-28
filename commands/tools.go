// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/version"

	"github.com/juju/1.25-upgrade/juju2/api"
	coretools "github.com/juju/1.25-upgrade/juju2/tools"
)

const toolsURLTemplate = "https://%s/tools/%s-%s"

func newToolsWrangler(conn api.Connection) *toolsWrangler {
	return &toolsWrangler{
		conn:   conn,
		client: utils.GetNonValidatingHTTPClient(),
	}
}

type toolsWrangler struct {
	conn   api.Connection
	client *http.Client
}

func (w *toolsWrangler) version() string {
	version, ok := w.conn.ServerVersion()
	if !ok {
		panic("can't download tools without logging into controller")
	}
	return version.String()
}

func (w *toolsWrangler) getTools(seriesArch string) error {
	toolsURL := fmt.Sprintf(toolsURLTemplate, w.conn.Addr(), w.version(), seriesArch)
	toolsVersion := version.MustParseBinary(w.version() + "-" + seriesArch)

	// Look to see if the directory is already there, if it is, assume
	// that it is good.
	downloadedToolsDir := path.Join(toolsDir, toolsVersion.String())
	if _, err := os.Stat(downloadedToolsDir); err == nil {
		logger.Infof("%s exists\n", downloadedToolsDir)
		return nil
	}

	logger.Infof("Downloading tools: %s\n", toolsURL)
	resp, err := w.client.Get(toolsURL)
	if err != nil {
		return errors.Annotatef(err, "downloading tools %s", toolsVersion)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("bad HTTP response: %v", resp.Status)
	}

	err = UnpackTools(toolsDir, toolsVersion, resp.Body)
	if err != nil {
		return errors.Errorf("cannot unpack tools: %v", err)
	}
	return nil

}

// UnpackTools reads a set of juju tools in gzipped tar-archive
// format and unpacks them into the appropriate tools directory
// within dataDir. If a valid tools directory already exists,
// UnpackTools returns without error.
func UnpackTools(dataDir string, toolsVersion version.Binary, r io.Reader) (err error) {
	// Unpack the gzip file and compute the checksum.
	zr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer zr.Close()
	f, err := ioutil.TempFile(os.TempDir(), "tools-tar")
	if err != nil {
		return err
	}
	_, err = io.Copy(f, zr)
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	// Make a temporary directory in the tools directory,
	// first ensuring that the tools directory exists.
	dir, err := ioutil.TempDir(toolsDir, "unpacking-")
	if err != nil {
		return err
	}
	defer removeAll(dir)

	// Checksum matches, now reset the file and untar it.
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.ContainsAny(hdr.Name, "/\\") {
			return fmt.Errorf("bad name %q in tools archive", hdr.Name)
		}
		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("bad file type %c in file %q in tools archive", hdr.Typeflag, hdr.Name)
		}
		name := path.Join(dir, hdr.Name)
		if err := writeFile(name, os.FileMode(hdr.Mode&0777), tr); err != nil {
			return errors.Annotatef(err, "tar extract %q failed", name)
		}
	}
	// Write some metadata about the tools.
	tools := &coretools.Tools{Version: toolsVersion}
	toolsMetadataData, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(dir, toolsFile), toolsMetadataData, 0644)
	if err != nil {
		return err
	}

	// The tempdir is created with 0700, so we need to make it more
	// accessable for juju-run.
	err = os.Chmod(dir, 0755)
	if err != nil {
		return err
	}

	return os.Rename(dir, path.Join(toolsDir, toolsVersion.String()))
}

func writeFile(name string, mode os.FileMode, r io.Reader) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
