// The `azure` plugin for SHIELD is intended to be a back-end storage
// plugin, wrapping Azure's Blobstore Service.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following
// SHIELD Job components:
//
//  Target: no
//  Store:  yes
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to determine
// how to connect to Azure Blobstore, and where to place/retrieve the data once connected.
// your endpoint JSON should look something like this:
//
//    {
//        "storage_account":       "your-access-key-id",
//        "storage_account_key":   "your-secret-access-key",
//        "storage_container":     "storage-container-name",
//    }
//
// STORE DETAILS
//
// When storing data, this plugin connects to the Azure Blobstore, and uploads
// the data into the specified container, using a filename with the following format:
// into the specified storage container, using a path/filename with the following format:
//
//    <YYYY>-<MM>-<DD>-<HH-mm-SS>-<UUID>
//
// Upon successful storage, the plugin then returns this filename to SHIELD to use
// as the `store_key` when the data needs to be retrieved, or purged.
//
// If the storage container does not exist, it will be auto-created for you.
//
// RETRIEVE DETAILS
//
// When retrieving data, this plugin connects to the Azure Blobstore, and retrieves the data
// located in the specified storage container, identified by the `store_key` provided by SHIELD.
//
// PURGE DETAILS
//
// When purging data, this plugin connects to the Azure Blobstore, and deletes the data
// located in the specified storage container, identified by the `store_key` provided by SHIELD.
//
// DEPENDENCIES
//
// None.
//
package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	azure "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/starkandwayne/goutils/ansi"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := AzurePlugin{
		Name:    "Azure Blobstore Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},

		Example: `
{
  "storage_account":     "your-access-key-id",
  "storage_account_key": "your-secret-access-key",
  "storage_container":   "storage-container-name",
}
`,
		Defaults: `
{
  # there are no defaults.
  # all keys are required.
}
`,
	}

	plugin.Run(p)
}

type AzurePlugin plugin.PluginInfo

type AzureConnectionInfo struct {
	StorageAccount    string
	StorageAccountKey string
	StorageContainer  string
}

func (p AzurePlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p AzurePlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("storage_account")
	if err != nil {
		ansi.Printf("@R{\u2717 storage_account     %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 storage_account}     @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("storage_account_key", "")
	if err != nil {
		ansi.Printf("@R{\u2717 storage_account_key %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 storage_account_key} @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("storage_container")
	if err != nil {
		ansi.Printf("@R{\u2717 storage_container   %s}\n", err)
		fail = true
	} else {
		containerFail := false
		containerValidator := regexp.MustCompile(`^[a-z0-9\-]+$`)
		if !containerValidator.MatchString(s) {
			fail = true
			containerFail = true
			ansi.Printf("@R{\u2717 storage_container   invalid characters (must be lower-case alpha-numeric plus dash)}\n")
		}
		if len(s) < 3 || len(s) > 63 {
			fail = true
			containerFail = true
			ansi.Printf("@R{\u2717 storage_container   is too long/short (must be 3-63 characters)}\n")
		}

		if !containerFail {
			ansi.Printf("@G{\u2713 storage_container}   @C{%s}\n", s)
		}
	}

	if fail {
		return fmt.Errorf("azure: invalid configuration")
	}
	return nil
}

func (p AzurePlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p AzurePlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p AzurePlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	az, err := getAzureConnInfo(endpoint)
	if err != nil {
		return "", err
	}
	client, err := az.Connect()
	if err != nil {
		return "", err
	}

	created, err := client.CreateContainerIfNotExists(az.StorageContainer, azure.ContainerAccessTypePrivate)
	if err != nil {
		return "", err
	}
	if created {
		plugin.DEBUG("Created new storage container: %s", az.StorageContainer)
	}

	path := az.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	plugin.DEBUG("Creating new backup blob: %s", path)
	err = client.PutAppendBlob(az.StorageContainer, path, nil)
	if err != nil {
		return "", err
	}

	uploaded := 0
	for {
		buf := make([]byte, 4*1024*1024)
		n, err := io.ReadFull(os.Stdin, buf)
		plugin.DEBUG("Uploading %d bytes for a total of %d", n, uploaded)
		if n > 0 {
			uploaded += n
			err := client.AppendBlock(az.StorageContainer, path, buf[:n], nil)
			if err != nil {
				return "", err
			}
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return "", err
		}
	}
	plugin.DEBUG("Successfully uploaded %d bytes of data", uploaded)

	return path, nil
}

func (p AzurePlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	az, err := getAzureConnInfo(endpoint)
	if err != nil {
		return err
	}
	client, err := az.Connect()
	if err != nil {
		return err
	}

	reader, err := client.GetBlob(az.StorageContainer, file)
	if err != nil {
		return err
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		return err
	}

	err = reader.Close()
	if err != nil {
		return err
	}

	return nil
}

func (p AzurePlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	az, err := getAzureConnInfo(endpoint)
	if err != nil {
		return err
	}
	client, err := az.Connect()
	if err != nil {
		return err
	}

	return client.DeleteBlob(az.StorageContainer, file, nil)
}

func getAzureConnInfo(e plugin.ShieldEndpoint) (AzureConnectionInfo, error) {
	storageAcct, err := e.StringValue("storage_account")
	if err != nil {
		return AzureConnectionInfo{}, err
	}

	storageAcctKey, err := e.StringValue("storage_account_key")
	if err != nil {
		return AzureConnectionInfo{}, err
	}

	storageContainer, err := e.StringValue("storage_container")
	if err != nil {
		return AzureConnectionInfo{}, err
	}

	return AzureConnectionInfo{
		StorageAccount:    storageAcct,
		StorageAccountKey: storageAcctKey,
		StorageContainer:  storageContainer,
	}, nil
}

func (az AzureConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%04d-%02d-%02d-%02d%02d%02d-%s", year, mon, day, hour, min, sec, uuid)
	return path
}

func (az AzureConnectionInfo) Connect() (azure.BlobStorageClient, error) {
	client, err := azure.NewBasicClient(az.StorageAccount, az.StorageAccountKey)
	if err != nil {
		return azure.BlobStorageClient{}, err
	}

	return client.GetBlobService(), nil
}
