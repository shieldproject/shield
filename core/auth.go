package core

import (
	"fmt"
)

func (core *Core) FindAuthProvider(identifier string) (AuthProvider, error) {
	var provider AuthProvider

	for _, auth := range core.auth {
		if auth.Identifier == identifier {
			switch auth.Backend {
			case "github":
				provider = &GithubAuthProvider{
					Name:       auth.Name,
					Identifier: identifier,
					core:       core,
				}
			case "uaa":
				provider = &UAAAuthProvider{
					Identifier: identifier,
					core:       core,
				}
			default:
				return nil, fmt.Errorf("unrecognized auth provider type '%s'", auth.Backend)
			}

			if err := provider.Configure(auth.Properties); err != nil {
				return nil, fmt.Errorf("failed to configure '%s' auth provider '%s': %s",
					auth.Backend, auth.Identifier, err)
			}
			return provider, nil
		}
	}

	return nil, fmt.Errorf("auth provider %s not defined", identifier)
}
