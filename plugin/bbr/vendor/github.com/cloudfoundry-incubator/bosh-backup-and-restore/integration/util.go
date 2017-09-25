package integration

import (
	"fmt"
	"os/exec"

	"time"

	"io/ioutil"

	"archive/tar"
	"crypto/sha256"
	"io"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v2"
)

type Binary struct {
	path       string
	runTimeout time.Duration
}

func NewBinary(path string) Binary {
	return Binary{path: path, runTimeout: 99999 * time.Hour}
}

func (b Binary) Start(cwd string, env []string, params ...string) (*gexec.Session, io.WriteCloser) {
	command := exec.Command(b.path, params...)
	command.Env = env
	command.Dir = cwd
	stdin, err := command.StdinPipe()
	Expect(err).ToNot(HaveOccurred())
	fmt.Fprintf(GinkgoWriter, "Running command: %v %v in %s with env %v\n", b.path, params, cwd, env)
	fmt.Fprintf(GinkgoWriter, "Command output start\n")
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session, stdin
}

func (b Binary) Run(cwd string, env []string, params ...string) *gexec.Session {
	session, _ := b.Start(cwd, env, params...)
	Eventually(session, b.runTimeout).Should(gexec.Exit())
	fmt.Fprintf(GinkgoWriter, "Command output end\n")
	fmt.Fprintf(GinkgoWriter, "Exited with %d\n", session.ExitCode())
	return session
}

type instanceMetadata struct {
	InstanceName  string                   `yaml:"name"`
	InstanceIndex string                   `yaml:"index"`
	Artifacts     []customArtifactMetadata `yaml:"artifacts"`
}

func (metadata instanceMetadata) FindArtifact(artifactName string) customArtifactMetadata {
	for _, artifact := range metadata.Artifacts {
		if artifact.Name == artifactName {
			return artifact
		}
	}
	Fail("No artifact with " + artifactName + " found")
	return customArtifactMetadata{}
}

type customArtifactMetadata struct {
	Name      string            `yaml:"name"`
	Checksums map[string]string `yaml:"checksums"`
}

type backupActivityMetadata struct {
	StartTime  string `yaml:"start_time"`
	FinishTime string `yaml:"finish_time"`
}

type metadata struct {
	InstancesMetadata       []instanceMetadata       `yaml:"instances"`
	CustomArtifactsMetadata []customArtifactMetadata `yaml:"custom_artifacts,omitempty"`
	BackupActivityMetadata  backupActivityMetadata   `yaml:"backup_activity"`
}

func ParseMetadata(filePath string) metadata {
	metadataContents := metadata{}
	contents, _ := ioutil.ReadFile(filePath)
	yaml.Unmarshal(contents, &metadataContents)
	return metadataContents
}

func ShaFor(contents string) string {
	shasum := sha256.New()
	shasum.Write([]byte(contents))
	return fmt.Sprintf("%x", shasum.Sum(nil))
}

type TarArchive struct {
	path string
}

func OpenTarArchive(path string) TarArchive {
	return TarArchive{path: path}
}

func (t TarArchive) Files() []string {
	reader := getTarReader(t.path)

	filenames := []string{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() {
			filenames = append(filenames, info.Name())
		}
	}
	return filenames
}

func (t TarArchive) FileContents(fileName string) string {
	reader := getTarReader(t.path)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() && info.Name() == fileName {
			contents, err := ioutil.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			return string(contents)
		}
	}
	Fail("File " + fileName + " not found in tar " + t.path)
	return ""
}

func getTarReader(path string) *tar.Reader {
	fileReader, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())
	return tar.NewReader(fileReader)
}
