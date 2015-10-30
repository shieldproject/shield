//line spec.y:2
package main

import __yyfmt__ "fmt"

//line spec.y:2
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
)

type TimeInHHMM struct {
	hours   uint8
	minutes uint8
}

//line spec.y:21
type yySymType struct {
	yys    int
	time   *TimeInHHMM
	number uint8
}

const NUMBER = 57346
const ORDINAL = 57347
const DAILY = 57348
const WEEKLY = 57349
const MONTHLY = 57350
const AT = 57351
const ON = 57352
const AM = 57353
const PM = 57354
const EVERYDAY = 57355
const SUNDAY = 57356
const MONDAY = 57357
const TUESDAY = 57358
const WEDNESDAY = 57359
const THURSDAY = 57360
const FRIDAY = 57361
const SATURDAY = 57362

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"NUMBER",
	"ORDINAL",
	"DAILY",
	"WEEKLY",
	"MONTHLY",
	"AT",
	"ON",
	"AM",
	"PM",
	"EVERYDAY",
	"SUNDAY",
	"MONDAY",
	"TUESDAY",
	"WEDNESDAY",
	"THURSDAY",
	"FRIDAY",
	"SATURDAY",
	"':'",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line spec.y:91

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
}

func LexerForFile(filename string) *yyLex {
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("ERROR! %s\n", err)
		return nil
	}
	l := &yyLex{buf: s}

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

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 29
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 69

var yyAct = [...]int{

	19, 8, 44, 31, 32, 33, 42, 22, 24, 26,
	32, 33, 28, 20, 30, 36, 20, 27, 39, 29,
	48, 25, 34, 20, 35, 41, 37, 4, 38, 40,
	11, 12, 13, 14, 15, 16, 17, 20, 43, 3,
	45, 2, 23, 1, 47, 46, 10, 5, 7, 9,
	50, 49, 0, 0, 6, 11, 12, 13, 14, 15,
	16, 17, 20, 20, 0, 0, 0, 21, 18,
}
var yyPact = [...]int{

	41, -1000, -1000, -1000, -1000, 59, 58, 33, 12, 8,
	16, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 19, -1000,
	-7, 19, -1000, 19, 5, 19, -1000, 19, 9, -1000,
	21, -1000, -1000, -1000, -1000, -4, 16, -1000, -8, 19,
	-1000, -1, 16, -1000, 46, -1000, -1000, -1000, -1000, -1000,
	-1000,
}
var yyPgo = [...]int{

	0, 0, 3, 43, 41, 39, 27, 1, 20,
}
var yyR1 = [...]int{

	0, 3, 3, 3, 4, 4, 4, 4, 1, 1,
	1, 5, 5, 5, 5, 2, 2, 7, 7, 7,
	7, 7, 7, 7, 6, 6, 6, 8, 8,
}
var yyR2 = [...]int{

	0, 1, 1, 1, 3, 2, 3, 2, 3, 4,
	2, 5, 4, 3, 2, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 5, 4, 3, 1, 1,
}
var yyChk = [...]int{

	-1000, -3, -4, -5, -6, 6, 13, 7, -7, 8,
	5, 14, 15, 16, 17, 18, 19, 20, 9, -1,
	4, 9, -1, 9, -1, 9, -1, 9, -7, -1,
	21, -2, 11, 12, -1, -1, 10, -1, -1, 9,
	-1, 4, 10, -7, 10, -1, -2, -7, -8, 5,
	4,
}
var yyDef = [...]int{

	0, -2, 1, 2, 3, 0, 0, 0, 0, 0,
	0, 17, 18, 19, 20, 21, 22, 23, 0, 5,
	0, 0, 7, 0, 0, 0, 14, 0, 0, 4,
	0, 10, 15, 16, 6, 0, 0, 13, 0, 0,
	26, 8, 0, 12, 0, 25, 9, 11, 24, 27,
	28,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 21,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lookahead func() int
}

func (p *yyParserImpl) Lookahead() int {
	return p.lookahead()
}

func yyNewParser() yyParser {
	p := &yyParserImpl{
		lookahead: func() int { return -1 },
	}
	return p
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yytoken := -1 // yychar translated into internal numbering
	yyrcvr.lookahead = func() int { return yychar }
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yychar = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar, yytoken = yylex1(yylex, &yylval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yychar = -1
		yytoken = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar, yytoken = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yychar = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 8:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line spec.y:58
		{
			yyVAL.time = &TimeInHHMM{hours: yyDollar[1].number, minutes: yyDollar[3].number}
		}
	case 9:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line spec.y:59
		{
			yyVAL.time = &TimeInHHMM{hours: yyDollar[1].number + yyDollar[4].number, minutes: yyDollar[3].number}
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line spec.y:60
		{
			yyVAL.time = &TimeInHHMM{hours: yyDollar[1].number + yyDollar[2].number, minutes: 0}
		}
	case 15:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line spec.y:69
		{
			yyVAL.number = 0
		}
	case 16:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line spec.y:70
		{
			yyVAL.number = 12
		}
	}
	goto yystack /* stack new state and value */
}
