package db

import "fmt"

//ErrExists is the error that should be returned from a db function if an item
// could not be inserted because it already exists in the database
type ErrExists struct {
	message string
}

//NewErrExists makes a new ErrExists object... works a hell of a lot like fmt.Errorf
func NewErrExists(format string, args ...interface{}) error {
	return ErrExists{
		message: fmt.Sprintf(format, args...),
	}
}

func (e ErrExists) Error() string {
	return e.message
}
