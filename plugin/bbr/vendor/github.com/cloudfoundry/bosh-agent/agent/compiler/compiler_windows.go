package compiler

import (
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func (c concreteCompiler) runPackagingCommand(compilePath, enablePath string, pkg Package) error {
	command := boshsys.Command{
		Name: "powershell",
		Args: []string{"-command", fmt.Sprintf(`"iex (get-content -raw %s)"`, PackagingScriptName)},
		Env: map[string]string{
			"BOSH_COMPILE_TARGET":  compilePath,
			"BOSH_INSTALL_TARGET":  enablePath,
			"BOSH_PACKAGE_NAME":    pkg.Name,
			"BOSH_PACKAGE_VERSION": pkg.Version,
		},
		WorkingDir: compilePath,
	}

	_, err := c.runner.RunCommand("compilation", PackagingScriptName, command)
	if err != nil {
		return bosherr.WrapError(err, "Running packaging script")
	}
	return nil
}
