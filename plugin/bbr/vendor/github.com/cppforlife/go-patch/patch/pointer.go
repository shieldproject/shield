package patch

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	rfc6901Decoder = strings.NewReplacer("~0", "~", "~1", "/")
	rfc6901Encoder = strings.NewReplacer("~", "~0", "/", "~1")
)

// More or less based on https://tools.ietf.org/html/rfc6901
type Pointer struct {
	tokens []Token
}

func MustNewPointerFromString(str string) Pointer {
	ptr, err := NewPointerFromString(str)
	if err != nil {
		panic(err.Error())
	}

	return ptr
}

func NewPointerFromString(str string) (Pointer, error) {
	tokens := []Token{RootToken{}}

	if len(str) == 0 {
		return Pointer{tokens}, nil
	}

	if !strings.HasPrefix(str, "/") {
		return Pointer{}, fmt.Errorf("Expected to start with '/'")
	}

	tokenStrs := strings.Split(str, "/")
	tokenStrs = tokenStrs[1:]

	optional := false

	for i, tok := range tokenStrs {
		isLast := i == len(tokenStrs)-1

		tok = rfc6901Decoder.Replace(tok)

		// parse as after last index
		if isLast && tok == "-" {
			tokens = append(tokens, AfterLastIndexToken{})
			continue
		}

		// parse as index
		idx, err := strconv.Atoi(tok)
		if err == nil {
			tokens = append(tokens, IndexToken{idx})
			continue
		}

		if strings.HasSuffix(tok, "?") {
			optional = true
		}

		// parse name=val
		kv := strings.SplitN(tok, "=", 2)
		if len(kv) == 2 {
			token := MatchingIndexToken{
				Key:      kv[0],
				Value:    strings.TrimSuffix(kv[1], "?"),
				Optional: optional,
			}

			tokens = append(tokens, token)
			continue
		}

		// it's a map key
		token := KeyToken{
			Key:      strings.TrimSuffix(tok, "?"),
			Optional: optional,
		}

		tokens = append(tokens, token)
	}

	return Pointer{tokens}, nil
}

func NewPointer(tokens []Token) Pointer {
	if len(tokens) == 0 {
		panic("Expected at least one token")
	}

	_, ok := tokens[0].(RootToken)
	if !ok {
		panic("Expected first token to be root")
	}

	return Pointer{tokens}
}

func (p Pointer) Tokens() []Token { return p.tokens }

func (p Pointer) IsSet() bool { return len(p.tokens) > 0 }

func (p Pointer) String() string {
	var strs []string

	optional := false

	for _, token := range p.tokens {
		switch typedToken := token.(type) {
		case RootToken:
			strs = append(strs, "")

		case IndexToken:
			strs = append(strs, fmt.Sprintf("%d", typedToken.Index))

		case AfterLastIndexToken:
			strs = append(strs, "-")

		case MatchingIndexToken:
			key := rfc6901Encoder.Replace(typedToken.Key)
			val := rfc6901Encoder.Replace(typedToken.Value)

			if typedToken.Optional {
				if !optional {
					val += "?"
					optional = true
				}
			}

			strs = append(strs, fmt.Sprintf("%s=%s", key, val))

		case KeyToken:
			str := rfc6901Encoder.Replace(typedToken.Key)

			if typedToken.Optional { // /key?/key2/key3
				if !optional {
					str += "?"
					optional = true
				}
			}

			strs = append(strs, str)

		default:
			panic(fmt.Sprintf("Unknown token type '%T'", typedToken))
		}
	}

	return strings.Join(strs, "/")
}

// UnmarshalFlag satisfies go-flags flag interface
func (p *Pointer) UnmarshalFlag(data string) error {
	ptr, err := NewPointerFromString(data)
	if err != nil {
		return err
	}

	*p = ptr

	return nil
}
