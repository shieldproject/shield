package supervisor

import (
	"github.com/google/go-github/github"
	"github.com/markbates/goth"
	"github.com/starkandwayne/goutils/log"
	"net/http"
)

var OAuthVerifier MembershipChecker

type MembershipChecker interface {
	Verify(goth.User, *http.Client) bool
}

// implement github verifier that takes users, groups, teams, retrieves membership from github library, checks to see if they're configured

type GithubVerifier struct {
	Orgs []string
}

func (gv *GithubVerifier) Verify(user goth.User, c *http.Client) bool {
	// If none specified, just let any authenticated person in
	if len(gv.Orgs) == 0 {
		log.Debugf("No orgs specified for authorization, denying access to '%s'.", user.NickName)
		return false
	}

	ghc := github.NewClient(c)

	page := 1
	var orgs []string

	for page != 0 {
		o, r, err := ghc.Organizations.List("", &github.ListOptions{Page: page})

		if err != nil {
			log.Debugf("Error retrieving org info for '%s' from GitHub, cannot authorize them", user.NickName)
			return false
		}

		for _, org := range o {
			orgs = append(orgs, *org.Login)
		}

		page = r.NextPage
	}

	log.Debugf("User orgs: %#v", orgs)
	log.Debugf("Allowed Orgs: %#v", gv.Orgs)

	for _, target := range gv.Orgs {
		log.Debugf("Seeing if '%s' is in GitHub Org '%s'", user.NickName, target)
		for _, org := range orgs {
			if org == target {
				log.Debugf("'%s' is an allowed org, granting access to '%s'", target, user.NickName)
				return true
			}
		}
	}
	return false
}
