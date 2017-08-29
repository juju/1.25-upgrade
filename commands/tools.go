// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

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
		cache:  make(map[string]*coretools.Tools),
	}
}

type toolsWrangler struct {
	conn   api.Connection
	client *http.Client
	cache  map[string]*coretools.Tools
}

func (tw *toolsWrangler) version() version.Number {
	version, ok := tw.conn.ServerVersion()
	if !ok {
		panic("can't download tools without logging into controller")
	}
	return version
}

func (tw *toolsWrangler) binary(seriesArch string) version.Binary {
	return version.MustParseBinary(tw.version().String() + "-" + seriesArch)
}

func (tw *toolsWrangler) url(seriesArch string) string {
	return fmt.Sprintf(toolsURLTemplate, tw.conn.Addr(), tw.version(), seriesArch)
}

func (tw *toolsWrangler) getTools(seriesArch string) error {
	toolsURL := tw.url(seriesArch)
	toolsVersion := tw.binary(seriesArch)

	// Look to see if the file is already there, if it is, assume
	// that it is good.
	downloadedTools := toolsFilePath(tw.version(), seriesArch)
	if _, err := os.Stat(downloadedTools); err == nil {
		logger.Infof("%s exists\n", downloadedTools)
		return nil
	}

	logger.Infof("Downloading tools: %s\n", toolsURL)
	resp, err := tw.client.Get(toolsURL)
	if err != nil {
		return errors.Annotatef(err, "downloading tools %s", toolsVersion)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("bad HTTP response: %v", resp.Status)
	}

	err = writeFile(downloadedTools, 0644, resp.Body)
	if err != nil {
		return errors.Errorf("cannot save tools: %v", err)
	}
	return nil
}

func (tw *toolsWrangler) metadata(seriesArch string) (*coretools.Tools, error) {
	if cached, ok := tw.cache[seriesArch]; ok {
		return cached, nil
	}
	err := tw.getTools(seriesArch)
	if err != nil {
		return nil, errors.Trace(err)
	}
	toolsFile := toolsFilePath(tw.version(), seriesArch)
	info, err := os.Stat(toolsFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	hash := sha256.New()
	f, err := os.Open(toolsFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer f.Close()
	_, err = io.Copy(hash, f)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &coretools.Tools{
		Version: tw.binary(seriesArch),
		URL:     tw.url(seriesArch),
		Size:    info.Size(),
		SHA256:  hex.EncodeToString(hash.Sum(nil)),
	}, nil
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

func toolsFilePath(ver version.Number, seriesArch string) string {
	return path.Join(toolsDir, fmt.Sprintf("%s-%s.tgz", ver, seriesArch))
}
