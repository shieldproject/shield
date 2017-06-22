package core

var Problems = map[string]string{
	"legacy-shield-agent-version": "This SHIELD agent is not reporting its version, which means that it is probably a v6.x version of SHIELD.  It will not be able to report back health and status information to this SHIELD Core.  Similarly, plugin metadata will be unavailable for this agent, and SHIELD operators and site administrators will have to operate without it for all targets that use this agent for backup and restore operations.",

	"dev-shield-agent-version": "This SHIELD agent is reporting its version as 'dev', which makes it difficult to determine its exact featureset.  Dev builds of SHIELD are not recommended for production.",
}
