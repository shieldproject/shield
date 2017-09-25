package types

import (
	"crypto/rand"
	"math/big"
)

type passwordGenerator struct {
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

const DefaultPasswordLength = 20

func NewPasswordGenerator() ValueGenerator {
	return passwordGenerator{}
}

func (passwordGenerator) Generate(parameters interface{}) (interface{}, error) {

	lengthLetterRunes := big.NewInt(int64(len(letterRunes)))
	passwordRunes := make([]rune, DefaultPasswordLength)

	for i := range passwordRunes {
		index, err := rand.Int(rand.Reader, lengthLetterRunes)
		if err != nil {
			return nil, err
		}

		passwordRunes[i] = letterRunes[index.Int64()]
	}

	return string(passwordRunes), nil
}
