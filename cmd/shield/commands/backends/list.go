package backends

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List configured SHIELD backends
var List = &commands.Command{
	Summary: "List configured SHIELD backend aliases",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "full", Desc: "Display verbose information about all backends",
			},
		},
		JSONOutput: `[{
			"name":"mybackend",
			"uri":"https://10.244.2.2:443",
			"skip_ssl_validation":false
		}]`,
	},
	RunFn: cliListBackends,
	Group: commands.BackendsGroup,
}

func cliListBackends(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'backends' command")

	backends := config.List()
	sort.Slice(backends, func(i, j int) bool { return backends[i].Name < backends[j].Name })

	if *opts.Raw {
		internal.RawJSON(backends)
		return nil
	}

	var t *tui.Table
	if *opts.Full {
		var err error
		t, err = verboseList(backends)
		if err != nil {
			return err
		}
	} else {
		t = conciseList(backends)
	}

	t.Output(os.Stdout)

	return nil
}

func conciseList(backends []*api.Backend) *tui.Table {
	t := tui.NewTable("Name", "Backend URI")

	for _, backend := range backends {
		isCurrent := config.Current() != nil && backend.Name == config.Current().Name

		if backend.SkipSSLValidation {
			backend.Name = ansi.Sprintf("%s @R{(insecure)}", backend.Name)
		}

		if isCurrent {
			backend.Name = ansi.Sprintf("@G{%s}", backend.Name)
		}

		t.Row(backend, backend.Name, backend.Address)
	}

	return &t
}

func verboseList(backends []*api.Backend) (*tui.Table, error) {
	t := tui.NewTable("Name", "Backend URI", "Insecure", "Token", "CA Cert")

	for _, backend := range backends {
		isCurrent := config.Current() != nil && backend.Name == config.Current().Name

		isInsecure := fmt.Sprintf("%t", backend.SkipSSLValidation)

		if backend.CACert != "" {
			var err error
			backend.CACert, err = certInfoString(backend.CACert)
			if err != nil {
				return nil, err
			}
		}

		if isCurrent {
			backend.Name = ansi.Sprintf("@G{%s}", backend.Name)
		}

		if backend.SkipSSLValidation {
			isInsecure = ansi.Sprintf("@R{%s}", isInsecure)
		}

		t.Row(backend, backend.Name, backend.Address, isInsecure, backend.Token, backend.CACert)
	}

	return &t, nil
}

func certInfoString(pemcert string) (string, error) {
	block, rest := pem.Decode([]byte(pemcert))
	if block == nil {
		return "", fmt.Errorf("Failed to decode PEM block")
	}
	if len(strings.TrimSpace(string(rest))) > 0 {
		return "", fmt.Errorf("Extra contents found in cert (is this a cert bundle?)")
	}
	if block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("PEM Block type is not CERTIFICATE")
	}

	//Check that this is a well-formatted cert
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("PEM Block does not contain a valid certificate: %s", err.Error())
	}

	ipstrings := []string{}
	for _, ip := range cert.IPAddresses {
		ipstrings = append(ipstrings, ip.String())
	}

	sans := append(cert.DNSNames, cert.EmailAddresses...)
	sans = append(sans, ipstrings...)

	now := time.Now()
	valid := (now.Equal(cert.NotBefore) || now.After(cert.NotBefore)) &&
		(now.Equal(cert.NotAfter) || now.Before(cert.NotAfter))

	timeFormat := "01/02/06 15:04"
	validityRange := fmt.Sprintf("%s - %s", cert.NotBefore.Format(timeFormat), cert.NotAfter.Format(timeFormat))

	subjectStr := formatSubject(cert.Subject)

	if valid {
		return ansi.Sprintf("@Y{Subject:} %s\n@Y{Valid During:} %s\n@Y{SANs:} %s",
			subjectStr, validityRange, strings.Join(sans, ", ")), nil
	}
	return ansi.Sprintf("@Y{Subject:} %s\n@Y{Valid During:} @R{%s}\n@Y{SANs:} %s",
		subjectStr, validityRange, strings.Join(sans, ", ")), nil
}

func formatSubject(name pkix.Name) string {
	ss := []string{}
	if name.CommonName != "" {
		ss = append(ss, fmt.Sprintf("cn=%s", name.CommonName))
	}
	for _, s := range name.Country {
		ss = append(ss, fmt.Sprintf("c=%s", s))
	}
	for _, s := range name.Province {
		ss = append(ss, fmt.Sprintf("st=%s", s))
	}
	for _, s := range name.Locality {
		ss = append(ss, fmt.Sprintf("l=%s", s))
	}
	for _, s := range name.Organization {
		ss = append(ss, fmt.Sprintf("o=%s", s))
	}
	for _, s := range name.OrganizationalUnit {
		ss = append(ss, fmt.Sprintf("ou=%s", s))
	}

	return strings.Join(ss, ",")
}
