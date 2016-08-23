package supervisor

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/markbates/goth"
	"github.com/starkandwayne/goutils/botta"
	"github.com/starkandwayne/goutils/log"
)

var OAuthVerifier MembershipChecker

type MembershipChecker interface {
	Membership(goth.User, *http.Client) (map[string]interface{}, error)
	Verify(string, map[string]interface{}) bool
}

// implement github verifier that takes users, groups, teams, retrieves membership from github library, checks to see if they're configured

type GithubVerifier struct {
	Orgs []string
}

func (gv *GithubVerifier) Membership(user goth.User, c *http.Client) (map[string]interface{}, error) {
	ghc := github.NewClient(c)

	page := 1
	var orgs []string

	for page != 0 {
		o, r, err := ghc.Organizations.List("", &github.ListOptions{Page: page})
		if err != nil {
			return nil, fmt.Errorf("Error retrieving org info for '%s' from GitHub, cannot authorize them", user.NickName)
		}

		for _, org := range o {
			orgs = append(orgs, *org.Login)
		}

		page = r.NextPage
	}
	return map[string]interface{}{"Orgs": orgs}, nil
}

func (gv *GithubVerifier) Verify(user string, membership map[string]interface{}) bool {
	// If none specified, don't let anyone in
	if len(gv.Orgs) == 0 {
		log.Debugf("No orgs specified for authorization, denying access to '%s'.", user)
		return false
	}

	log.Debugf("User orgs: %#v", membership["Orgs"])
	log.Debugf("Allowed Orgs: %#v", gv.Orgs)

	for _, target := range gv.Orgs {
		log.Debugf("Seeing if '%s' is in GitHub Org '%s'", user, target)

		var orgs []string
		var ok bool
		orgs, ok = membership["Orgs"].([]string)
		if !ok {
			os, ok := membership["Orgs"].([]interface{})
			if ok {
				for _, o := range os {
					s, ok := o.(string)
					if !ok {
						log.Debugf("Unexpected data type for group: %#v", o)
						return false
					}
					orgs = append(orgs, s)
				}
			} else {
				log.Debugf("Unexpected data type for groups: %#v", membership["Orgs"])
				return false
			}
		}

		for _, org := range orgs {
			if org == target {
				log.Debugf("'%s' is an allowed org, granting access to '%s'", target, user)
				return true
			}
		}
	}
	return false
}

type UAAVerifier struct {
	Groups []string
	UAA    string
}

func (uv *UAAVerifier) Verify(user string, membership map[string]interface{}) bool {
	// If none specified, don't let anyone in
	if len(uv.Groups) == 0 {
		log.Debugf("No groups specified for authorization, denying access to '%s'.", user)
		return false
	}

	log.Debugf("User Groups: %#v", membership["Groups"])
	log.Debugf("Allowed Groups: %#v", uv.Groups)

	for _, target := range uv.Groups {
		log.Debugf("Seeing if '%s' is in UAA Group '%s'", user, target)

		var groups []string
		var ok bool
		groups, ok = membership["Groups"].([]string)
		if !ok {
			g, ok := membership["Groups"].([]interface{})
			if ok {
				for _, o := range g {
					s, ok := o.(string)
					if !ok {
						log.Debugf("Unexpected data type for group: %#v", o)
						return false
					}
					groups = append(groups, s)
				}
			} else {
				log.Debugf("Unexpected data type for groups: %#v", membership["Groups"])
				return false
			}
		}

		for _, group := range groups {
			if group == target {
				log.Debugf("'%s' is an allowed group, granting access to '%s'", target, user)
				return true
			}
		}
	}
	log.Debugf("No groups matched")
	return false
}

func (uv *UAAVerifier) Membership(user goth.User, c *http.Client) (map[string]interface{}, error) {
	var filterGroups []string
	for _, g := range uv.Groups {
		filterGroups = append(filterGroups, fmt.Sprintf("displayName=%%22%s%%22", g))
	}
	filter := strings.Join(filterGroups, "+or+")
	botta.SetClient(c)
	req, err := botta.Get(fmt.Sprintf("%s/Groups?attributes=displayName,members&filter=%s", uv.UAA, filter))
	if err != nil {
		return nil, err
	}

	resp, err := botta.Issue(req)
	if err != nil {
		return nil, err
	}

	groups, err := resp.ArrayVal("resources")
	if err != nil {
		return nil, err
	}

	var membership []string
	for _, g := range groups {
		group, ok := g.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Unexpected data type for group returned from UAA: %#v", g)
		}

		name, ok := group["displayName"].(string)
		if !ok {
			return nil, fmt.Errorf("Unexpected data type for group name returned form UAA: %#v", g)
		}

		members, ok := group["members"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("Unexpected data type for group membership returned from UAA: %#v", group["members"])
		}

		for _, m := range members {
			member, ok := m.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Unexpected data type for group member returned from UAA: %#v", m)
			}

			uid, ok := member["value"].(string)
			if !ok {
				return nil, fmt.Errorf("Unexpected data type for group member uid returned from UAA: %#v", member)
			}
			if uid == user.UserID {
				membership = append(membership, name)
			}
		}
	}
	return map[string]interface{}{"Groups": membership}, nil
}

// FIXME: tests.. ghalrdfsa
