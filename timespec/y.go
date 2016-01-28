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
const HOURLY = 57348
const DAILY = 57349
const WEEKLY = 57350
const MONTHLY = 57351
const AT = 57352
const ON = 57353
const AM = 57354
const PM = 57355
const HALF = 57356
const QUARTER = 57357
const AFTER = 57358
const TIL = 57359
const EVERYDAY = 57360
const EVERYHOUR = 57361
const SUNDAY = 57362
const MONDAY = 57363
const TUESDAY = 57364
const WEDNESDAY = 57365
const THURSDAY = 57366
const FRIDAY = 57367
const SATURDAY = 57368

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"NUMBER",
	"ORDINAL",
	"HOURLY",
	"DAILY",
	"WEEKLY",
	"MONTHLY",
	"AT",
	"ON",
	"AM",
	"PM",
	"HALF",
	"QUARTER",
	"AFTER",
	"TIL",
	"EVERYDAY",
	"EVERYHOUR",
	"SUNDAY",
	"MONDAY",
	"TUESDAY",
	"WEDNESDAY",
	"THURSDAY",
	"FRIDAY",
	"SATURDAY",
	"'h'",
	"'H'",
	"'x'",
	"'X'",
	"'*'",
	"':'",
	"' '",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line lang.y:121

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 25,
	1, 24,
	-2, 22,
}

const yyNprod = 55
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 154

var yyAct = [...]int{

	55, 66, 37, 49, 38, 57, 58, 57, 58, 12,
	23, 25, 38, 40, 42, 44, 46, 34, 69, 35,
	72, 33, 32, 24, 47, 54, 56, 71, 82, 50,
	51, 57, 58, 48, 27, 28, 29, 30, 31, 53,
	68, 67, 59, 1, 60, 52, 63, 77, 64, 74,
	70, 6, 62, 68, 67, 5, 4, 73, 15, 16,
	17, 18, 19, 20, 21, 3, 78, 79, 38, 2,
	75, 76, 80, 81, 45, 25, 38, 26, 0, 84,
	0, 22, 43, 85, 83, 33, 32, 0, 68, 67,
	14, 7, 9, 11, 13, 65, 0, 0, 27, 28,
	29, 30, 31, 10, 8, 15, 16, 17, 18, 19,
	20, 21, 25, 38, 38, 61, 0, 0, 0, 41,
	39, 0, 33, 32, 15, 16, 17, 18, 19, 20,
	21, 0, 0, 0, 0, 27, 28, 29, 30, 31,
	15, 16, 17, 18, 19, 20, 21, 38, 0, 0,
	0, 0, 0, 36,
}
var yyPact = [...]int{

	85, -1000, -1000, -1000, -1000, -1000, -1000, 71, 7, 143,
	110, 109, 72, 64, 120, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 108, -1000, -29, -1000, 13, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 108, -1000, 0, -1000, -7, 0,
	-1000, 0, 104, 0, -1000, 0, 84, 8, -1000, 23,
	-1000, -1000, -1000, -1000, 16, -1000, 19, -1000, -1000, -1000,
	38, 120, -1000, -1000, 36, 49, -1000, -1000, -1000, 0,
	-1000, -1000, -5, -1000, 120, -1000, -1000, 49, -1000, -1000,
	-1000, -1000, 19, -1000, -1000, -1000,
}
var yyPgo = [...]int{

	0, 10, 2, 0, 1, 77, 9, 69, 65, 56,
	55, 51, 43, 23,
}
var yyR1 = [...]int{

	0, 12, 7, 7, 7, 7, 8, 8, 8, 8,
	9, 9, 9, 9, 13, 13, 13, 13, 13, 13,
	5, 5, 5, 1, 1, 1, 1, 2, 2, 2,
	2, 2, 10, 10, 10, 10, 10, 10, 3, 3,
	6, 6, 6, 6, 6, 6, 6, 11, 11, 11,
	11, 11, 11, 4, 4,
}
var yyR2 = [...]int{

	0, 1, 1, 1, 1, 1, 3, 2, 3, 2,
	3, 2, 3, 2, 0, 1, 1, 1, 1, 1,
	1, 1, 1, 3, 1, 2, 2, 3, 4, 5,
	2, 3, 5, 4, 4, 3, 3, 2, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 5, 4, 4,
	3, 4, 3, 1, 1,
}
var yyChk = [...]int{

	-1000, -12, -7, -8, -9, -10, -11, 6, 19, 7,
	18, 8, -6, 9, 5, 20, 21, 22, 23, 24,
	25, 26, 10, -1, -13, 4, -5, 27, 28, 29,
	30, 31, 15, 14, 10, -1, 10, -2, 4, 10,
	-2, 10, -2, 10, -2, 10, -2, -6, -1, 32,
	16, 17, -1, -2, 32, -3, 33, 12, 13, -2,
	-2, 11, -6, -2, -2, 11, -4, 5, 4, 10,
	-2, 4, 4, -3, 11, -6, -6, 11, -4, -4,
	-2, -3, 33, -6, -4, -3,
}
var yyDef = [...]int{

	0, -2, 1, 2, 3, 4, 5, 14, 14, 0,
	0, 0, 0, 0, 0, 40, 41, 42, 43, 44,
	45, 46, 14, 7, 0, -2, 0, 15, 16, 17,
	18, 19, 20, 21, 14, 9, 0, 11, 0, 0,
	13, 0, 0, 0, 37, 0, 0, 0, 6, 0,
	25, 26, 8, 10, 0, 30, 0, 38, 39, 12,
	0, 0, 35, 36, 0, 0, 50, 53, 54, 0,
	52, 23, 27, 31, 0, 34, 33, 0, 49, 48,
	51, 28, 0, 32, 47, 29,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 33, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 31, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 32, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 28, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 30, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 27, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	29,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26,
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
		//line lang.y:48
		{
			yylex.(*yyLex).spec = yyDollar[1].spec
		}
	case 6:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:56
		{
			yyVAL.spec = hourly(yyDollar[3].time)
		}
	case 7:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:57
		{
			yyVAL.spec = hourly(yyDollar[2].time)
		}
	case 8:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:58
		{
			yyVAL.spec = hourly(yyDollar[3].time)
		}
	case 9:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:59
		{
			yyVAL.spec = hourly(yyDollar[2].time)
		}
	case 10:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:62
		{
			yyVAL.spec = daily(yyDollar[3].time)
		}
	case 11:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:63
		{
			yyVAL.spec = daily(yyDollar[2].time)
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:64
		{
			yyVAL.spec = daily(yyDollar[3].time)
		}
	case 13:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:65
		{
			yyVAL.spec = daily(yyDollar[2].time)
		}
	case 20:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:70
		{
			yyVAL.numval = 15
		}
	case 21:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:71
		{
			yyVAL.numval = 30
		}
	case 22:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:72
		{
			yyVAL.numval = yyDollar[1].numval
		}
	case 23:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:75
		{
			yyVAL.time = hhmm(0, yyDollar[3].numval)
		}
	case 24:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:76
		{
			yyVAL.time = hhmm(0, yyDollar[1].numval)
		}
	case 25:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:77
		{
			yyVAL.time = hhmm(0, yyDollar[1].numval)
		}
	case 26:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:78
		{
			yyVAL.time = hhmm(0, 60-yyDollar[1].numval)
		}
	case 27:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:81
		{
			yyVAL.time = hhmm(yyDollar[1].numval, yyDollar[3].numval)
		}
	case 28:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:82
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[4].numval, yyDollar[3].numval)
		}
	case 29:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line lang.y:83
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[5].numval, yyDollar[3].numval)
		}
	case 30:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:84
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[2].numval, 0)
		}
	case 31:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:85
		{
			yyVAL.time = hhmm(yyDollar[1].numval+yyDollar[3].numval, 0)
		}
	case 32:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line lang.y:88
		{
			yyVAL.spec = weekly(yyDollar[3].time, yyDollar[5].wday)
		}
	case 33:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:89
		{
			yyVAL.spec = weekly(yyDollar[2].time, yyDollar[4].wday)
		}
	case 34:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:90
		{
			yyVAL.spec = weekly(yyDollar[3].time, yyDollar[4].wday)
		}
	case 35:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:91
		{
			yyVAL.spec = weekly(yyDollar[2].time, yyDollar[3].wday)
		}
	case 36:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:92
		{
			yyVAL.spec = weekly(yyDollar[3].time, yyDollar[1].wday)
		}
	case 37:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line lang.y:93
		{
			yyVAL.spec = weekly(yyDollar[2].time, yyDollar[1].wday)
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:96
		{
			yyVAL.numval = 0
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:97
		{
			yyVAL.numval = 12
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:100
		{
			yyVAL.wday = time.Sunday
		}
	case 41:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:101
		{
			yyVAL.wday = time.Monday
		}
	case 42:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:102
		{
			yyVAL.wday = time.Tuesday
		}
	case 43:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:103
		{
			yyVAL.wday = time.Wednesday
		}
	case 44:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:104
		{
			yyVAL.wday = time.Thursday
		}
	case 45:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:105
		{
			yyVAL.wday = time.Friday
		}
	case 46:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line lang.y:106
		{
			yyVAL.wday = time.Saturday
		}
	case 47:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line lang.y:109
		{
			yyVAL.spec = mday(yyDollar[3].time, yyDollar[5].numval)
		}
	case 48:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:110
		{
			yyVAL.spec = mday(yyDollar[2].time, yyDollar[4].numval)
		}
	case 49:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:111
		{
			yyVAL.spec = mday(yyDollar[3].time, yyDollar[4].numval)
		}
	case 50:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:112
		{
			yyVAL.spec = mday(yyDollar[2].time, yyDollar[3].numval)
		}
	case 51:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line lang.y:113
		{
			yyVAL.spec = mweek(yyDollar[4].time, yyDollar[2].wday, yyDollar[1].numval)
		}
	case 52:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line lang.y:114
		{
			yyVAL.spec = mweek(yyDollar[3].time, yyDollar[2].wday, yyDollar[1].numval)
		}
	}
	goto yystack /* stack new state and value */
}
