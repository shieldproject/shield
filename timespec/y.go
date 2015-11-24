//line lang.y:2
package timespec

import __yyfmt__ "fmt"

//line lang.y:2
import (
	"time"
)

//line lang.y:9
type yySymType struct {
	yys    int
	numval uint
	time   int
	wday   time.Weekday
	spec   *Spec
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
	"' '",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line lang.y:95

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 37
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 106

var yyAct = [...]int{

	33, 20, 44, 9, 35, 36, 35, 36, 21, 23,
	25, 27, 29, 46, 45, 30, 32, 34, 49, 54,
	21, 31, 35, 36, 37, 47, 38, 1, 41, 40,
	42, 21, 48, 59, 5, 50, 28, 4, 46, 45,
	46, 45, 52, 53, 43, 55, 56, 21, 21, 57,
	58, 21, 26, 24, 21, 60, 22, 61, 3, 19,
	62, 11, 6, 8, 10, 2, 0, 0, 0, 7,
	12, 13, 14, 15, 16, 17, 18, 51, 0, 0,
	0, 12, 13, 14, 15, 16, 17, 18, 39, 0,
	0, 0, 12, 13, 14, 15, 16, 17, 18, 12,
	13, 14, 15, 16, 17, 18,
}
var yyPact = [...]int{

	56, -1000, -1000, -1000, -1000, -1000, 50, 47, 44, 43,
	27, 85, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 4,
	-1000, -5, 4, -1000, 4, 78, 4, -1000, 4, 34,
	16, -1000, 14, -1000, -7, -1000, -1000, -1000, 67, 85,
	-1000, -1000, 9, 36, -1000, -1000, -1000, 4, -1000, 11,
	-1000, 85, -1000, -1000, 36, -1000, -1000, -1000, -1000, -7,
	-1000, -1000, -1000,
}
var yyPgo = [...]int{

	0, 1, 0, 2, 3, 65, 58, 37, 34, 27,
}
var yyR1 = [...]int{

	0, 9, 5, 5, 5, 6, 6, 6, 6, 1,
	1, 1, 1, 1, 7, 7, 7, 7, 7, 7,
	2, 2, 4, 4, 4, 4, 4, 4, 4, 8,
	8, 8, 8, 8, 8, 3, 3,
}
var yyR2 = [...]int{

	0, 1, 1, 1, 1, 3, 2, 3, 2, 3,
	4, 5, 2, 3, 5, 4, 4, 3, 3, 2,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 5,
	4, 4, 3, 4, 3, 1, 1,
}
var yyChk = [...]int{

	-1000, -9, -5, -6, -7, -8, 6, 13, 7, -4,
	8, 5, 14, 15, 16, 17, 18, 19, 20, 9,
	-1, 4, 9, -1, 9, -1, 9, -1, 9, -1,
	-4, -1, 21, -2, 22, 11, 12, -1, -1, 10,
	-4, -1, -1, 10, -3, 5, 4, 9, -1, 4,
	-2, 10, -4, -4, 10, -3, -3, -1, -2, 22,
	-4, -3, -2,
}
var yyDef = [...]int{

	0, -2, 1, 2, 3, 4, 0, 0, 0, 0,
	0, 0, 22, 23, 24, 25, 26, 27, 28, 0,
	6, 0, 0, 8, 0, 0, 0, 19, 0, 0,
	0, 5, 0, 12, 0, 20, 21, 7, 0, 0,
	17, 18, 0, 0, 32, 35, 36, 0, 34, 9,
	13, 0, 16, 15, 0, 31, 30, 33, 10, 0,
	14, 29, 11,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 22, 3, 3, 3, 3, 3, 3, 3,
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

	case 1:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:41
		{
			yylex.(*yyLex).spec = yyDollar[1].spec
		}
	case 5:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:49
		{
			yyVAL.spec = daily(yyDollar[3].time)
		}
	case 6:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:50
		{
			yyVAL.spec = daily(yyDollar[2].time)
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:51
		{
			yyVAL.spec = daily(yyDollar[3].time)
		}
	case 8:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:52
		{
			yyVAL.spec = daily(yyDollar[2].time)
		}
	case 9:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:55
		{
			yyVAL.time = hhmm(yyDollar[1].numval, yyDollar[3].numval)
		}
	case 10:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:56
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[4].numval, yyDollar[3].numval)
		}
	case 11:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line lang.y:57
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[5].numval, yyDollar[3].numval)
		}
	case 12:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:58
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[2].numval, 0)
		}
	case 13:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:59
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[3].numval, 0)
		}
	case 14:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line lang.y:62
		{
			yyVAL.spec = weekly(yyDollar[3].time, yyDollar[5].wday)
		}
	case 15:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:63
		{
			yyVAL.spec = weekly(yyDollar[2].time, yyDollar[4].wday)
		}
	case 16:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:64
		{
			yyVAL.spec = weekly(yyDollar[3].time, yyDollar[4].wday)
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:65
		{
			yyVAL.spec = weekly(yyDollar[2].time, yyDollar[3].wday)
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:66
		{
			yyVAL.spec = weekly(yyDollar[3].time, yyDollar[1].wday)
		}
	case 19:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:67
		{
			yyVAL.spec = weekly(yyDollar[2].time, yyDollar[1].wday)
		}
	case 20:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:70
		{
			yyVAL.numval = 0
		}
	case 21:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:71
		{
			yyVAL.numval = 12
		}
	case 22:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:74
		{
			yyVAL.wday = time.Sunday
		}
	case 23:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:75
		{
			yyVAL.wday = time.Monday
		}
	case 24:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:76
		{
			yyVAL.wday = time.Tuesday
		}
	case 25:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:77
		{
			yyVAL.wday = time.Wednesday
		}
	case 26:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:78
		{
			yyVAL.wday = time.Thursday
		}
	case 27:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:79
		{
			yyVAL.wday = time.Friday
		}
	case 28:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:80
		{
			yyVAL.wday = time.Saturday
		}
	case 29:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line lang.y:83
		{
			yyVAL.spec = mday(yyDollar[3].time, yyDollar[5].numval)
		}
	case 30:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:84
		{
			yyVAL.spec = mday(yyDollar[2].time, yyDollar[4].numval)
		}
	case 31:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:85
		{
			yyVAL.spec = mday(yyDollar[3].time, yyDollar[4].numval)
		}
	case 32:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:86
		{
			yyVAL.spec = mday(yyDollar[2].time, yyDollar[3].numval)
		}
	case 33:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:87
		{
			yyVAL.spec = mweek(yyDollar[4].time, yyDollar[2].wday, yyDollar[1].numval)
		}
	case 34:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:88
		{
			yyVAL.spec = mweek(yyDollar[3].time, yyDollar[2].wday, yyDollar[1].numval)
		}
	}
	goto yystack /* stack new state and value */
}
