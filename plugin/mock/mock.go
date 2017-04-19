package main

import (
	fmt "github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/plugin"
	"io"
	"os"
)

func main() {
	p := MockPlugin{
		Name:    "Mock Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "yes",
		},
		Example: `
{
  "string1": "example",   # required, no default
  "string2": "optional",  # defaults to "not set"

  "float1":  1.234,       # required, no default
  "float2":  2.468,       # defaults to 42.0

  "bool1":   true,        # required, no default
  "bool2":   false,       # defaults to true

  "list":    [1,2,3],     # optional, default empty
  "map":     {...},       # optional, default empty
}
`,
		Defaults: `
{
  "string2": "not set",
  "float2":  42.0,
  "bool2":   true
}
`,
	}

	plugin.Run(p)
}

type MockPlugin plugin.PluginInfo

func (p MockPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p MockPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		err  error
		fail bool

		b bool
		s string
		f float64
		l []interface{}
		m map[string]interface{}
	)
	s, err = endpoint.StringValue("string1")
	if err != nil {
		fmt.Printf("@R{\u2717 string1              %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 string1}              @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("string2", "not set")
	if err != nil {
		fmt.Printf("@R{\u2717 string2              %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 string2}              @C{%s}\n", s)
	}

	f, err = endpoint.FloatValue("float1")
	if err != nil {
		fmt.Printf("@R{\u2717 float1               %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 float1}               @C{%v}\n", f)
	}

	f, err = endpoint.FloatValueDefault("float2", 42.0)
	if err != nil {
		fmt.Printf("@R{\u2717 float2               %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 float2}               @C{%v}\n", f)
	}

	b, err = endpoint.BooleanValue("bool1")
	if err != nil {
		fmt.Printf("@R{\u2717 bool1                %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bool1}                @C{%v}\n", b)
	}

	b, err = endpoint.BooleanValueDefault("bool2", true)
	if err != nil {
		fmt.Printf("@R{\u2717 bool2                %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bool2}                @C{%v}\n", b)
	}

	l, err = endpoint.ArrayValue("list")
	if err != nil {
		fmt.Printf("@R{\u2717 list                 %s}\n", err)
		fail = true
	} else {
		for i, v := range l {
			fmt.Printf("@G{\u2713 list}                 @C{[%d] = %v}\n", i, v)
		}
	}

	m, err = endpoint.MapValue("map")
	if err != nil {
		fmt.Printf("@R{\u2717 map                  %s}\n", err)
		fail = true
	} else {
		for k, v := range m {
			fmt.Printf("@G{\u2713 map}                  @C{'%s' = %v}\n", k, v)
		}
	}

	if fail {
		return fmt.Errorf("mock plugin: invalid configuration")
	}
	return nil
}

func Nowhere() io.Writer {
	out, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open /dev/null: %s\n", err)
		os.Exit(99)
	}
	return out
}

func (p MockPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	fmt.Fprintf(os.Stdout, "mock data\n")
	return nil
}

func (p MockPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	_, err := io.Copy(Nowhere(), os.Stdin)
	return err
}

func (p MockPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "fake-storage-key", nil
}

func (p MockPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	fmt.Fprintf(os.Stdout, "mock backup\n")
	return nil
}

func (p MockPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return nil
}
