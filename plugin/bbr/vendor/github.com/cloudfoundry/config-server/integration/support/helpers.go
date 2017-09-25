package support

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const AssetsDir string = "assets"
const ServerStartTimeout int = 10
const StartScript string = "./start_server.sh"
const StopScript string = "./stop_server.sh"
const DbSetupScript string = "./setup_db.sh"

var HTTPSClient = createHTTPSClient()

func UnmarshalJSONString(requestBody io.ReadCloser) map[string]interface{} {
	var f interface{}

	if err := json.NewDecoder(requestBody).Decode(&f); err != nil {
		panic("String provided cannot be decoded as JSON")
	}

	return f.(map[string]interface{})
}

func ParseCertString(certString string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certString))
	crt, err := x509.ParseCertificate(block.Bytes)
	return crt, err
}

func ValidToken() string {
	tokenPath := pathForAsset("uaa.token")
	dat, err := ioutil.ReadFile(tokenPath)

	if err != nil {
		panic(err.Error())
	}

	return string(dat)
}

func StartServer() {
	err := exec.Command(StartScript).Start()
	if err != nil {
		fmt.Println("Failed to start Config Server: ", err.Error())
	}
	waitForServerToStart()
}

func StopServer() {
	err := exec.Command(StopScript).Start()
	if err != nil {
		fmt.Println("Failed to start Config Server: ", err.Error())
	}

	waitForServerToStop()
}

func SetupDB() {
	db := os.Getenv("DB")
	err := exec.Command(DbSetupScript, db).Run()
	if err != nil {
		panic("Failed to setup DB: " + err.Error())
	}
}

func waitForServerToStart() {
	for i := 0; i < ServerStartTimeout; i++ {
		resp, err := SendGetRequestByID("1")

		if err == nil && resp.StatusCode == 404 {
			break
		}

		if i == ServerStartTimeout-1 {
			panic("Could not start config server in " + string(ServerStartTimeout) + " seconds")
		}

		time.Sleep(time.Second)
	}
}

func waitForServerToStop() {
	for i := 0; i < ServerStartTimeout; i++ {
		_, err := SendGetRequestByName("some_name")

		if err != nil {
			break
		}

		if i == ServerStartTimeout-1 {
			panic("Could not stop config server in " + string(ServerStartTimeout) + " seconds")
		}

		time.Sleep(time.Second)
	}
}

func pathForAsset(fileName string) string {
	var path, rootDir string

	rootDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err.Error())
	}

	path = filepath.Join(rootDir, AssetsDir, fileName)

	return path
}

func createHTTPSClient() *http.Client {
	sslCertPath := pathForAsset("ssl.crt")
	sslKeyPath := pathForAsset("ssl.key")
	rootCAPath := pathForAsset("ssl_root_ca.crt")

	cert, err := tls.LoadX509KeyPair(sslCertPath, sslKeyPath)
	if err != nil {
		panic(err.Error())
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(rootCAPath)
	if err != nil {
		panic(err.Error())
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	client := &http.Client{Transport: transport}

	return client
}
