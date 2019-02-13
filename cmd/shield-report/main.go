package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jhunt/go-cli"
)

var opt struct {
	Help bool `cli:"-h, --help"`

	Compression string `cli:"-c, --compression"`
}

func main() {
	if _, _, err := cli.Parse(&opt); err != nil {
		fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to parse command-line flags: %s\n", err)
		os.Exit(2)
	}

	if opt.Help {
		fmt.Printf("echo '{\"some\":\"json\"}' | shield-report [OPTIONS]\n\n")
		fmt.Printf("OPTIONS\n\n")
		fmt.Printf("  -h, --help             Show this help screen.\n")
		fmt.Printf("  -c, --compression ...  Set the \"compression\" key in the output JSON.\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to read standard input: %s\n", err)
		os.Exit(3)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to parse JSON from standard input: %s\n", err)
		os.Exit(3)
	}

	if opt.Compression != "" {
		data["compression"] = opt.Compression
	}

	b, err = json.Marshal(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to encode output JSON: %s\n", err)
		os.Exit(4)
	}
	os.Stdout.Write(b)
	os.Exit(0)
}
