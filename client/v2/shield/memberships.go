package shield

func (c *Client) Invite(tenant *Tenant, role string, users []*User) (Response, error) {
	type invitee struct {
		UUID    string `json:"uuid"`
		Account string `json:"account"`
		Role    string `json:"role"`
	}

	var in struct {
		Users []invitee `json:"users"`
	}

	in.Users = make([]invitee, len(users))
	for i := range users {
		in.Users[i] = invitee{
			UUID:    users[i].UUID,
			Account: users[i].Account,
			Role:    role,
		}
	}

	var r Response
	return r, c.post("/v2/tenants/"+tenant.UUID+"/invite", in, &r)
}

func (c *Client) Banish(tenant *Tenant, users []*User) (Response, error) {
	type banished struct {
		UUID    string `json:"uuid"`
		Account string `json:"account"`
	}

	var in struct {
		Users []banished `json:"users"`
	}

	in.Users = make([]banished, len(users))
	for i := range users {
		in.Users[i] = banished{
			UUID:    users[i].UUID,
			Account: users[i].Account,
		}
	}

	var r Response
	return r, c.post("/v2/tenants/"+tenant.UUID+"/banish", in, &r)
}
