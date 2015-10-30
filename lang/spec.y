%{
package main

import (
	"fmt"
	"io/ioutil"
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

%token DAILY
%token WEEKLY
%token MONTHLY
%token AT
%token ON
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
             ;

weekly_spec : WEEKLY AT time_in_HHMM ON day_name
            | WEEKLY    time_in_HHMM ON day_name
            | day_name AT time_in_HHMM
            | day_name    time_in_HHMM
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

type yyLex struct {
	buf []byte
}

func (l *yyLex) Lex(lval *yySymType) int {
	return 0
}

func (l *yyLex) Error(e string) {
	fmt.Printf("ERROR: %s\n", e)
}

func main() {
	s, err := ioutil.ReadFile("hardcoded.timespec")
	if err != nil {
		fmt.Printf("ERROR! %s\n", err)
		return
	}

	lexer := &yyLex{ buf: s }
	fmt.Printf("starting...\n")
	yyParse(lexer)
	fmt.Printf("DONE\n")
}
