package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jhunt/go-cli"
)

var Version = ""

var opt struct {
	Help    bool `cli:"-h, --help"`
	Version bool `cli:"-v, --version"`

	Compression string `cli:"-c, --compression"`
}

func main() {
	_, args, err := cli.Parse(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err)
		os.Exit(1)
	}
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "!!! extra arguments found\n")
		os.Exit(1)
	}

	if opt.Help {
		fmt.Fprintf(os.Stderr, "shield-reporting - Pipeline worker (shield-pipe) for reporting\n\n")
		fmt.Printf("Options\n")
		fmt.Printf("  -h, --help             Show this help screen.\n")
		fmt.Printf("  -v, --version          Display the SHIELD version.\n")
		fmt.Printf("\n")
		fmt.Printf("  -c, --compression x    Set the \"compression\" key in the output JSON.\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	if opt.Version {
		if Version == "" || Version == "dev" {
			fmt.Printf("shield-report (development)\n")
		} else {
			fmt.Printf("shield-report v%s\n", Version)
		}
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
	fmt.Printf("%s\n", string(b))
	os.Exit(0)
}
