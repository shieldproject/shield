%{
package timespec

import (
	"bytes"
	"io/ioutil"
	"time"
	"regexp"
	"strconv"
)

func hhmm(hours uint, minutes uint) int {
	for hours >= 24 {
		hours -= 12
	}
	return int(hours * 60 + minutes)
}
func daily(minutes int) *Spec {
	return &Spec{
		Interval:  Daily,
		TimeOfDay: minutes,
	}
}
func weekly(minutes int, weekday time.Weekday) *Spec {
	return &Spec{
		Interval:  Weekly,
		TimeOfDay: minutes,
		DayOfWeek: weekday,
	}
}
func mday(minutes int, day uint) *Spec {
	return &Spec{
		Interval:  Monthly,
		TimeOfDay: minutes,
		DayOfMonth: int(day),
	}
}
func mweek(minutes int, weekday time.Weekday, week uint) *Spec {
	return &Spec{
		Interval:  Monthly,
		TimeOfDay: minutes,
		DayOfWeek: weekday,
		Week:      int(week),
	}
}
%}

%union {
	numval  uint
	time    int
	wday    time.Weekday
	spec   *Spec
}

%type  <time>   time_in_HHMM
%type  <numval> am_or_pm month_day
%type  <wday>   day_name
%type  <spec>   spec daily_spec weekly_spec monthly_spec

%token <numval> NUMBER ORDINAL

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

timespec : spec {
                   yylex.(*yyLex).spec = $1
                }
         ;

spec : daily_spec | weekly_spec | monthly_spec
     ;

daily_spec : DAILY    AT time_in_HHMM  { $$ = daily($3) }
           | DAILY       time_in_HHMM  { $$ = daily($2) }
           | EVERYDAY AT time_in_HHMM  { $$ = daily($3) }
           | EVERYDAY    time_in_HHMM  { $$ = daily($2) }
           ;

time_in_HHMM : NUMBER ':' NUMBER          { $$ = hhmm($1,      $3) }
             | NUMBER ':' NUMBER am_or_pm { $$ = hhmm($1 + $4, $3) }
             | NUMBER am_or_pm            { $$ = hhmm($1 + $2, 0)  }
             ;

weekly_spec : WEEKLY AT time_in_HHMM ON day_name { $$ = weekly($3, $5) }
            | WEEKLY    time_in_HHMM ON day_name { $$ = weekly($2, $4) }
            | WEEKLY AT time_in_HHMM    day_name { $$ = weekly($3, $4) }
            | WEEKLY    time_in_HHMM    day_name { $$ = weekly($2, $3) }
            | day_name AT time_in_HHMM           { $$ = weekly($3, $1) }
            | day_name    time_in_HHMM           { $$ = weekly($2, $1) }
            ;

am_or_pm: AM { $$ = 0  }
        | PM { $$ = 12 }
        ;

day_name : SUNDAY     { $$ = time.Sunday    }
         | MONDAY     { $$ = time.Monday    }
         | TUESDAY    { $$ = time.Tuesday   }
         | WEDNESDAY  { $$ = time.Wednesday }
         | THURSDAY   { $$ = time.Thursday  }
         | FRIDAY     { $$ = time.Friday    }
         | SATURDAY   { $$ = time.Saturday  }
         ;

monthly_spec : MONTHLY AT time_in_HHMM ON month_day { $$ = mday($3, $5) }
             | MONTHLY    time_in_HHMM ON month_day { $$ = mday($2, $4) }
             | MONTHLY AT time_in_HHMM    month_day { $$ = mday($3, $4) }
             | MONTHLY    time_in_HHMM    month_day { $$ = mday($2, $3) }
             | ORDINAL day_name AT time_in_HHMM     { $$ = mweek($4, $2, $1) }
             | ORDINAL day_name    time_in_HHMM     { $$ = mweek($3, $2, $1) }
             ;

month_day: ORDINAL
         | NUMBER
         ;

%%

type Tokens struct {
	whitespace *regexp.Regexp
	number     *regexp.Regexp
	ordinal    *regexp.Regexp
}

type KeywordMatcher struct {
	token int
	match *regexp.Regexp
}

type yyLex struct {
	tokens   Tokens
	keywords []KeywordMatcher
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
	l.keywords = append(l.keywords, KeywordMatcher{token: DAILY, match: regexp.MustCompile(`^daily`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: WEEKLY, match: regexp.MustCompile(`^weekly`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: MONTHLY, match: regexp.MustCompile(`^monthly`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: AT, match: regexp.MustCompile(`^at`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: ON, match: regexp.MustCompile(`^on`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: AM, match: regexp.MustCompile(`^am`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: PM, match: regexp.MustCompile(`^pm`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: EVERYDAY, match: regexp.MustCompile(`^every\s+day`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: SUNDAY, match: regexp.MustCompile(`^sun(days?)?`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: MONDAY, match: regexp.MustCompile(`^mon(days?)?`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: TUESDAY, match: regexp.MustCompile(`^tue(s(days?)?)?`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: WEDNESDAY, match: regexp.MustCompile(`^wed(nesdays?)?`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: THURSDAY, match: regexp.MustCompile(`^thu(r(s(days?)?)?)?`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: FRIDAY, match: regexp.MustCompile(`^fri(days?)?`)})
	l.keywords = append(l.keywords, KeywordMatcher{token: SATURDAY, match: regexp.MustCompile(`^sat(urdays?)?`)})

	l.tokens.whitespace = regexp.MustCompile(`^\s+`)
	l.tokens.number = regexp.MustCompile(`^\d+`)
	l.tokens.ordinal = regexp.MustCompile(`^(\d+)(st|rd|nd|th)`)
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
