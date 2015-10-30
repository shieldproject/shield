%{
package main

import (
		"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
)

type TimeInHHMM struct {
	hours uint8
	minutes uint8
}

%}



%union {
	time *TimeInHHMM
	number uint8
}

%type <time> time_in_HHMM
%token <number> NUMBER
%token <number> ORDINAL
%type <number> am_or_pm

%token DAILY
%token WEEKLY
%token MONTHLY
%token AT
%token ON
%token AM
%token PM
%token EVERYDAY
%token SUNDAY
%token MONDAY
%token TUESDAY
%token WEDNESDAY
%token THURSDAY
%token FRIDAY
%token SATURDAY

%%

timespec : daily_spec | weekly_spec | monthly_spec
         ;

daily_spec : DAILY AT time_in_HHMM
           | DAILY    time_in_HHMM
           | EVERYDAY AT time_in_HHMM
           | EVERYDAY    time_in_HHMM
           ;

time_in_HHMM : NUMBER ':' NUMBER { $$ = &TimeInHHMM{ hours: $1, minutes: $3 } }
             | NUMBER ':' NUMBER am_or_pm { $$ = &TimeInHHMM{ hours: $1 + $4, minutes: $3 } }
             | NUMBER am_or_pm   { $$ = &TimeInHHMM{ hours: $1 + $2, minutes: 0 } }
             ;

weekly_spec : WEEKLY AT time_in_HHMM ON day_name
            | WEEKLY    time_in_HHMM ON day_name
            | day_name AT time_in_HHMM
            | day_name    time_in_HHMM
            ;

am_or_pm: AM { $$ = 0  }
        | PM { $$ = 12 }
        ;

day_name : SUNDAY
         | MONDAY
         | TUESDAY
         | WEDNESDAY
         | THURSDAY
         | FRIDAY
         | SATURDAY
         ;

monthly_spec : MONTHLY AT time_in_HHMM ON month_day
             | ORDINAL day_name AT time_in_HHMM
             | ORDINAL day_name    time_in_HHMM
             ;

month_day: ORDINAL
         | NUMBER
         ;

%%

type Tokens struct {
	whitespace *regexp.Regexp
	number *regexp.Regexp
	ordinal *regexp.Regexp
}

type KeywordMatcher struct {
	token int
	match *regexp.Regexp
}

type yyLex struct {
	tokens Tokens
	keywords []KeywordMatcher
	buf []byte
}

func LexerForFile(filename string) *yyLex {
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("ERROR! %s\n", err)
		return nil
	}
	l := &yyLex{ buf: s }

	l.keywords = append(l.keywords, KeywordMatcher{ token: DAILY,     match: regexp.MustCompile(`^daily`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: WEEKLY,    match: regexp.MustCompile(`^weekly`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: MONTHLY,   match: regexp.MustCompile(`^monthly`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: AT,        match: regexp.MustCompile(`^at`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: ON,        match: regexp.MustCompile(`^on`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: AM,        match: regexp.MustCompile(`^am`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: PM,        match: regexp.MustCompile(`^pm`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: EVERYDAY,  match: regexp.MustCompile(`^every\s+day`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: SUNDAY,    match: regexp.MustCompile(`^sun(days?)?`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: MONDAY,    match: regexp.MustCompile(`^mon(days?)?`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: TUESDAY,   match: regexp.MustCompile(`^tue(s(days?)?)?`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: WEDNESDAY, match: regexp.MustCompile(`^wed(nesdays?)?`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: THURSDAY,  match: regexp.MustCompile(`^thu(r(s(days?)?)?)?`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: FRIDAY,    match: regexp.MustCompile(`^fri(days?)?`) })
	l.keywords = append(l.keywords, KeywordMatcher{ token: SATURDAY,  match: regexp.MustCompile(`^sat(urdays?)?`) })

	l.tokens.whitespace = regexp.MustCompile(`^\s+`)
	l.tokens.number     = regexp.MustCompile(`^\d+`)
	l.tokens.ordinal    = regexp.MustCompile(`^(\d+)(st|rd|nd|th)`)

	return l
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

func numify(m []byte) uint8 {
	i, err := strconv.Atoi(stringify(m))
	if err != nil {
		panic("yo, stuff's broke")
	}
	return uint8(i)
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

	// number
	m = l.tokens.number.FindSubmatch(l.buf)
	if m != nil {
		l.eat(m)
		lval.number = numify(m[0])
		return NUMBER
	}

	// ordinal
	m = l.tokens.ordinal.FindSubmatch(l.buf)
	if m != nil {
		l.eat(m)
		lval.number = numify(m[1])
		return ORDINAL
	}

	if len(l.buf) == 0 {
		return 0
	}

	c := l.buf[0]
	l.buf = l.buf[1:]
	return int(c)
}

func (l *yyLex) Error(e string) {
	fmt.Printf("ERROR: %s\n", e)
}

func main() {
	fmt.Printf("starting...\n")
	lexer := LexerForFile("hardcoded.timespec")
	if lexer == nil {
		return
	}
	yyParse(lexer)
	fmt.Printf("DONE\n")
}
