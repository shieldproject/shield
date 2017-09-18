package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/markbates/goth/gothic"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/db"
	"golang.org/x/oauth2"
)

type userTenancyInfo struct {
	UUID uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
	Role string    `json:"role"`
}
type sessionedUserInfo struct {
	User struct {
		UUID    string `json:"uuid"`
		Name    string `json:"name"`
		Account string `json:"account"`
		Backend string `json:"backend"`
	} `json:"user"`

	Tenants []userTenancyInfo `json:"tenants"`
}

//getUserInfoFromRequestCookie will, given a request, validate the cookie present, and then return the sessioned user's information from the DB
func (core *Core) getUserInfoFromRequest(req *http.Request) (sessionedUserInfo, error) {
	cookie, err := req.Cookie("shield7")
	if err != nil {
		return sessionedUserInfo{}, err
	}

	user, err := core.DB.GetUserForSession(cookie.Value)
	if err != nil {
		return sessionedUserInfo{}, err
	}

	answer := sessionedUserInfo{}
	if user == nil {
		return sessionedUserInfo{}, err
	}

	answer.User.UUID = user.UUID.String()
	answer.User.Name = user.Name
	answer.User.Account = user.Account
	answer.User.Backend = user.Backend

	memberships, err := core.DB.GetMembershipsForUser(user.UUID)
	if err != nil {
		return sessionedUserInfo{}, nil
	}

	answer.Tenants = make([]userTenancyInfo, len(memberships))
	for i, membership := range memberships {
		answer.Tenants[i].UUID = membership.TenantUUID
		answer.Tenants[i].Name = membership.TenantName
		answer.Tenants[i].Role = membership.Role
	}
	if err != nil {
		return sessionedUserInfo{}, err
	}
	return answer, nil
}

func (core *Core) validateRight(userUUID uuid.UUID, tenantUUID uuid.UUID, requestedRight string) (bool, error) {
	role, err := core.DB.GetRoleForUserTenant(userUUID, tenantUUID)
	if err != nil {
		return false, err
	}

	if role.Right == "" {
		return false, nil
	}

	return role.Right == requestedRight, nil
}

func (core *Core) githubOauthHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)

	if err != nil {
		bailWithError(w, ClientErrorf(err.Error()))
		return
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: user.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	userInfo, _ := getGithubUserInfo(client)
	fmt.Println(userInfo)

	//Check if the user that logged in via github already exists
	dbUser, err := core.DB.GetUser(userInfo.Username, "github")
	if err != nil {
		bailWithError(w, ClientErrorf(err.Error()))
		return
	}
	if dbUser == nil {
		dbUser = &db.User{
			UUID:    uuid.NewRandom(),
			Name:    userInfo.Name,
			Account: userInfo.Username,
			Backend: "github",
			SysRole: "site_administrator",
			//FIXME we shouldn't be making everyone an admin
			//this is just for testing purposes
		}
		core.DB.CreateUser(dbUser)
	}
	session, err := core.createSession(dbUser)
	if err != nil {
		bailWithError(w, ClientErrorf(err.Error()))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "shield7",
		Value: session.UUID.String(),
		Path:  "/auth",
	})

	for org, team := range userInfo.OrgTeams {
		for _, currentTeam := range team {
			tenantUUID, role, err := core.DB.GetTenantRole(org, currentTeam)
			if err != nil {
				bailWithError(w, ClientErrorf(err.Error()))
				return
			}
			if tenantUUID != nil && role != "" {
				err = core.DB.AddUserToTenant(dbUser.UUID.String(), tenantUUID.String(), role)
				if err != nil {
					bailWithError(w, ClientErrorf(err.Error()))
					return
				}
			}

		}
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(302)
}

type GithubUserInfo struct {
	Username string
	Name     string
	OrgTeams map[string][]string
}

func getGithubUserInfo(githubClient *github.Client) (GithubUserInfo, error) {
	membership := make(map[string][]string)

	user, _, err := githubClient.Users.Get("")
	if err != nil {
		return GithubUserInfo{}, err
	}

	orgsList, _, err := githubClient.Organizations.List("", nil)
	if err != nil {
		return GithubUserInfo{}, err
	}

	for _, org := range orgsList {
		membership[*org.Login] = []string{}
	}

	teamList, _, err := githubClient.Organizations.ListUserTeams(nil)
	if err != nil {
		return GithubUserInfo{}, err
	}

	for _, team := range teamList {
		membership[*team.Organization.Login] = append(membership[*team.Organization.Login], *team.Name)
	}

	userInfo := GithubUserInfo{
		Username: *user.Login,
		Name:     *user.Name,
		OrgTeams: membership,
	}

	return userInfo, err
}
