package shield

func (c *Client) ChangePassword(oldpw, newpw string) (Response, error) {
	var r Response
	in := struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}{
		OldPassword: oldpw,
		NewPassword: newpw,
	}
	return r, c.post("/v2/auth/passwd", in, &r)
}
