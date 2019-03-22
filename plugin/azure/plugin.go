package main

import (
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	azure "github.com/Azure/azure-sdk-for-go/storage"
	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultPrefix = ""
)

func main() {
	p := AzurePlugin{
		Name:    "Microsoft Azure Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},

		Example: `
{
  "storage_account"     : "your-access-key-id",
  "storage_account_key" : "your-secret-access-key",
  "storage_container"   : "storage-container-name",

  "prefix"              : "/path/in/container",     # where to store archives, inside the container
}
`,
		Defaults: `
{
  # there are no defaults.
  # all keys are required.
}
`,

		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "store",
				Name:     "storage_account",
				Type:     "string",
				Title:    "Storage Account",
				Help:     "Name of the Azure Storage Account for accessing the blobstore.",
				Required: true,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "storage_account_key",
				Type:     "password",
				Title:    "Storage Account Key",
				Help:     "Secret Key of the Azure Storage Account for accessing the blobstore.",
				Required: true,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "storage_container",
				Type:     "string",
				Title:    "Storage container",
				Help:     "Name of the Container to store backup archives in.",
				Required: true,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "prefix",
				Type:  "string",
				Title: "Container Path Prefix",
				Help:  "An optional sub-path of the container to use for storing archives.  By default, archives are stored in the root of the container.",
			},
		},
	}

	plugin.Run(p)
}

type AzurePlugin plugin.PluginInfo

type AzureConnectionInfo struct {
	StorageAccount    string
	StorageAccountKey string
	StorageContainer  string
	PathPrefix        string
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
		fmt.Printf("@R{\u2717 storage_account     %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 storage_account}     @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("storage_account_key", "")
	if err != nil {
		fmt.Printf("@R{\u2717 storage_account_key %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 storage_account_key} @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValue("storage_container")
	if err != nil {
		fmt.Printf("@R{\u2717 storage_container   %s}\n", err)
		fail = true
	} else {
		containerFail := false
		containerValidator := regexp.MustCompile(`^[a-z0-9\-]+$`)
		if !containerValidator.MatchString(s) {
			fail = true
			containerFail = true
			fmt.Printf("@R{\u2717 storage_container   invalid characters (must be lower-case alpha-numeric plus dash)}\n")
		}
		if len(s) < 3 || len(s) > 63 {
			fail = true
			containerFail = true
			fmt.Printf("@R{\u2717 storage_container   is too long/short (must be 3-63 characters)}\n")
		}

		if !containerFail {
			fmt.Printf("@G{\u2713 storage_container}   @C{%s}\n", plugin.Redact(s))
		}
	}

	s, err = endpoint.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		fmt.Printf("@R{\u2717 prefix              %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 prefix}              (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		fmt.Printf("@G{\u2713 prefix}              @C{%s}\n", s)
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

func (p AzurePlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	az, err := getAzureConnInfo(endpoint)
	if err != nil {
		return "", 0, err
	}
	client, err := az.Connect()
	if err != nil {
		return "", 0, err
	}

	created, err := client.CreateContainerIfNotExists(az.StorageContainer, azure.ContainerAccessTypePrivate)
	if err != nil {
		return "", 0, err
	}
	if created {
		plugin.DEBUG("Created new storage container: %s", az.StorageContainer)
	}

	path := az.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	plugin.DEBUG("Creating new backup blob: %s", path)
	err = client.PutAppendBlob(az.StorageContainer, path, nil)
	if err != nil {
		return "", 0, err
	}

	var uploaded int64
	for {
		buf := make([]byte, 4*1024*1024)
		n, err := io.ReadFull(os.Stdin, buf)
		plugin.DEBUG("Uploading %d bytes for a total of %d", n, uploaded)
		if n > 0 {
			uploaded += int64(n)
			err := client.AppendBlock(az.StorageContainer, path, buf[:n], nil)
			if err != nil {
				return "", 0, err
			}
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return "", 0, err
		}
	}
	plugin.DEBUG("Successfully uploaded %d bytes of data", uploaded)

	return path, uploaded, nil
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

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return AzureConnectionInfo{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	return AzureConnectionInfo{
		StorageAccount:    storageAcct,
		StorageAccountKey: storageAcctKey,
		StorageContainer:  storageContainer,
		PathPrefix:        prefix,
	}, nil
}

func (az AzureConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%s/%04d-%02d-%02d-%02d%02d%02d-%s", az.PathPrefix, year, mon, day, hour, min, sec, uuid)
	// Remove double slashes
	path = strings.Replace(path, "//", "/", -1)
	return path
}

func (az AzureConnectionInfo) Connect() (azure.BlobStorageClient, error) {
	client, err := azure.NewBasicClient(az.StorageAccount, az.StorageAccountKey)
	if err != nil {
		return azure.BlobStorageClient{}, err
	}

	return client.GetBlobService(), nil
}
