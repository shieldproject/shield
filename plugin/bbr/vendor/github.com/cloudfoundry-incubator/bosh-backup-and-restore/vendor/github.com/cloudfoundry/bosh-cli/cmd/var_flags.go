package cmd

import (
	cfgtypes "github.com/cloudfoundry/config-server/types"

	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
)

// Shared
type VarFlags struct {
	VarKVs      []boshtpl.VarKV       `long:"var"        short:"v" value-name:"VAR=VALUE" description:"Set variable"`
	VarFiles    []boshtpl.VarFileArg  `long:"var-file"             value-name:"VAR=PATH"  description:"Set variable to file contents"`
	VarsFiles   []boshtpl.VarsFileArg `long:"vars-file"  short:"l" value-name:"PATH"      description:"Load variables from a YAML file"`
	VarsEnvs    []boshtpl.VarsEnvArg  `long:"vars-env"             value-name:"PREFIX"    description:"Load variables from environment variables (e.g.: 'MY' to load MY_var=value)"`
	VarsFSStore VarsFSStore           `long:"vars-store"           value-name:"PATH"      description:"Load/save variables from/to a YAML file"`
}

func (f VarFlags) AsVariables() boshtpl.Variables {
	var firstToUse []boshtpl.Variables

	firstToUse = append(firstToUse, f.kvsAsVars())

	for i, _ := range f.VarFiles {
		firstToUse = append(firstToUse, f.VarFiles[len(f.VarFiles)-i-1].Vars)
	}

	for i, _ := range f.VarsFiles {
		firstToUse = append(firstToUse, f.VarsFiles[len(f.VarsFiles)-i-1].Vars)
	}

	for i, _ := range f.VarsEnvs {
		firstToUse = append(firstToUse, f.VarsEnvs[len(f.VarsEnvs)-i-1].Vars)
	}

	store := &f.VarsFSStore

	if f.VarsFSStore.IsSet() {
		firstToUse = append(firstToUse, store)
	}

	vars := boshtpl.NewMultiVars(firstToUse)

	if f.VarsFSStore.IsSet() {
		store.ValueGeneratorFactory = cfgtypes.NewValueGeneratorConcrete(NewVarsCertLoader(vars))
	}

	return vars
}

func (f VarFlags) kvsAsVars() boshtpl.Variables {
	vars := boshtpl.StaticVariables{}

	for _, kv := range f.VarKVs {
		vars[kv.Name] = kv.Value
	}

	return vars
}
