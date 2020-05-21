package ssg

import (
	"io"

	ssgclient "github.com/jhunt/shield-storage-gateway/client"
	"github.com/shieldproject/shield/plugin"
)

func Run() {
	p := SsgPlugin{
		Name:    "SHIELD Storege Plugin",
		Author:  "SHIELD Core Team",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
	}
	plugin.Run(p)
}

type SsgPlugin plugin.PluginInfo

func (p SsgPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

type SsgConfig struct {
	Client        *ssgclient.Client
	UploadToken   string
	DownloadToken string
	UploadID      string
	DownloadID    string
	Path          string
}

func (p SsgPlugin) Validate(log io.Writer, endpoint plugin.ShieldEndpoint) error {
	return nil
}

func (p SsgPlugin) Backup(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p SsgPlugin) Restore(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func getSsgConfig(e plugin.ShieldEndpoint) (*SsgConfig, error) {
	url, err := e.StringValue("url")
	if err != nil {
		return nil, err
	}
	client := ssgclient.NewClient(url)

	uploadToken, err := e.StringValue("upload_token")
	if err != nil {
		return nil, err
	}

	downloadToken, err := e.StringValue("download_token")
	if err != nil {
		return nil, err
	}

	uploadID, err := e.StringValue("upload_id")
	if err != nil {
		return nil, err
	}

	downloadID, err := e.StringValue("download_id")
	if err != nil {
		return nil, err
	}

	path, err := e.StringValue("path")
	if err != nil {
		return nil, err
	}

	return &SsgConfig{
		Client:        client,
		UploadToken:   uploadToken,
		DownloadToken: downloadToken,
		UploadID:      uploadID,
		DownloadID:    downloadID,
		Path:          path,
	}, nil
}

func (p SsgPlugin) Store(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) (string, int64, error) {
	ssgConfig, err := getSsgConfig(endpoint)
	if err != nil {
		return "", 0, err
	}

	size, err := ssgConfig.Client.Upload(ssgConfig.UploadID, ssgConfig.UploadToken, in, true)
	if err != nil {
		return "", 0, err
	}
	return ssgConfig.Path, size, nil
}

func (p SsgPlugin) Retrieve(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint, file string) error {
	ssgConfig, err := getSsgConfig(endpoint)
	if err != nil {
		return err
	}

	plugin.Infof("retrieving backup archive\n"+
		"    from path '%s\n", file)

	in, err := ssgConfig.Client.Download(ssgConfig.DownloadID, ssgConfig.DownloadToken)
	if err != nil {
		return err
	}

	n, err := io.Copy(out, in)
	if err != nil {
		return err
	}
	in.Close()
	plugin.Infof("retrieved %d bytes of data", n)
	return nil
}

func (p SsgPlugin) Purge(log io.Writer, endpoint plugin.ShieldEndpoint, file string) error {
	return nil
}
