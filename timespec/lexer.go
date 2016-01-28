package timespec

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strconv"
)

type tokens struct {
	whitespace *regexp.Regexp
	number     *regexp.Regexp
	ordinal    *regexp.Regexp
}

type keywordMatcher struct {
	token int
	match *regexp.Regexp
}

type yyLex struct {
	tokens   tokens
	keywords []keywordMatcher
	buf      []byte
	spec     *Spec
}

func (l *yyLex) eat(m [][]byte) {
	l.buf = l.buf[len(m[0]):]
}

func stringify(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n < 0 {
		n = len(b)
	}

	return string(b[:n])
}

func numify(m []byte) uint {
	i, err := strconv.Atoi(stringify(m))
	if err != nil {
		panic("yo, stuff's broke")
	}
	return uint(i)
}

func (l *yyLex) init() {
	l.keywords = append(l.keywords, keywordMatcher{token: HOURLY, match: regexp.MustCompile(`(?i:^hourly)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: DAILY, match: regexp.MustCompile(`(?i:^daily)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: WEEKLY, match: regexp.MustCompile(`(?i:^weekly)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: MONTHLY, match: regexp.MustCompile(`(?i:^monthly)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: AT, match: regexp.MustCompile(`(?i:^at)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: ON, match: regexp.MustCompile(`(?i:^on)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: AM, match: regexp.MustCompile(`(?i:^am)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: PM, match: regexp.MustCompile(`(?i:^pm)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: HALF, match: regexp.MustCompile(`(?i:^half)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: QUARTER, match: regexp.MustCompile(`(?i:^quarter)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: AFTER, match: regexp.MustCompile(`(?i:^(past|after))`)})
	l.keywords = append(l.keywords, keywordMatcher{token: TIL, match: regexp.MustCompile(`(?i:^(un)?til)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: EVERYDAY, match: regexp.MustCompile(`(?i:^every\s+day)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: EVERYHOUR, match: regexp.MustCompile(`(?i:^every\s+hour)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: SUNDAY, match: regexp.MustCompile(`(?i:^sun(days?)?)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: MONDAY, match: regexp.MustCompile(`(?i:^mon(days?)?)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: TUESDAY, match: regexp.MustCompile(`(?i:^tue(s(days?)?)?)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: WEDNESDAY, match: regexp.MustCompile(`(?i:^wed(nesdays?)?)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: THURSDAY, match: regexp.MustCompile(`(?i:^thu(r(s(days?)?)?)?)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: FRIDAY, match: regexp.MustCompile(`(?i:^fri(days?)?)`)})
	l.keywords = append(l.keywords, keywordMatcher{token: SATURDAY, match: regexp.MustCompile(`(?i:^sat(urdays?)?)`)})

	l.tokens.whitespace = regexp.MustCompile(`^\s+`)
	l.tokens.number = regexp.MustCompile(`^\d+`)
	l.tokens.ordinal = regexp.MustCompile(`(?i:^(\d+)(st|rd|nd|th))`)
}

func LexerForString(s string) *yyLex {
	l := &yyLex{buf: []byte(s)}
	l.init()
	return l
}
func LexerForFile(filename string) *yyLex {
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil
	}
	l := &yyLex{buf: s}
	l.init()
	return l
}

func (l *yyLex) Lex(lval *yySymType) int {
	var m [][]byte

	// eat whitespace
	m = l.tokens.whitespace.FindSubmatch(l.buf)
	if m != nil {
		l.eat(m)
	}

	for _, keyword := range l.keywords {
		m = keyword.match.FindSubmatch(l.buf)
		if m != nil {
			l.eat(m)
			return keyword.token
		}
	}

	// ordinal
	m = l.tokens.ordinal.FindSubmatch(l.buf)
	if m != nil {
		l.eat(m)
		lval.numval = numify(m[1])
		return ORDINAL
	}

	// number
	m = l.tokens.number.FindSubmatch(l.buf)
	if m != nil {
		l.eat(m)
		lval.numval = numify(m[0])
		return NUMBER
	}

	if len(l.buf) == 0 {
		return 0
	}

	c := l.buf[0]
	l.buf = l.buf[1:]
	return int(c)
}

func (l *yyLex) Error(e string) {
}
