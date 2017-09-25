package server

type TokenValidator interface {
	Validate(token string) error
}
