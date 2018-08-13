%{
package timespec

import (
	"time"
)
%}

%union {
	numval  uint
	time    int
	wday    time.Weekday
	spec   *Spec
	truth   bool
}

%type  <time>   time_in_MM
%type  <time>   time_in_HHMM
%type  <numval> month_day nth_occurance_of minutes
%type  <wday>   day_name
%type  <spec>   spec hourly_spec daily_spec weekly_spec monthly_spec
%type  <truth>  am_or_pm

%token <numval> NUMBER ORDINAL

%token HOURLY
%token DAILY
%token WEEKLY
%token MONTHLY
%token FROM
%token AT
%token ON
%token AM
%token PM
%token HALF
%token EVERY
%token DAY
%token HOUR
%token QUARTER
%token AFTER
%token TIL
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

spec : hourly_spec | daily_spec | weekly_spec | monthly_spec
     ;

hourly_spec : HOURLY     AT time_in_MM             { $$ = hourly($3, 0) }
            | HOURLY        time_in_MM             { $$ = hourly($2, 0) }
            | EVERY QUARTER HOUR FROM time_in_MM   { $$ = hourly($5, 0.25) }
            | EVERY HALF HOUR FROM time_in_MM      { $$ = hourly($5, 0.5) }
            | EVERY HOUR AT time_in_MM             { $$ = hourly($4, 0) }
            | EVERY HOUR time_in_MM                { $$ = hourly($3, 0) }
            | EVERY NUMBER HOUR FROM time_in_HHMM  { $$ = hourly($5, float32($2)) }
            ;

daily_spec : DAILY    AT time_in_HHMM   { $$ = daily($3) }
           | DAILY       time_in_HHMM   { $$ = daily($2) }
           | EVERY DAY AT time_in_HHMM  { $$ = daily($4) }
           | EVERY DAY    time_in_HHMM  { $$ = daily($3) }
           ;

anyhour: | 'h' | 'H' | 'x' | 'X' | '*' ;

minutes: QUARTER { $$ = 15 }
       | HALF    { $$ = 30 }
       | NUMBER  { $$ = $1 }
       ;

time_in_MM: anyhour ':' NUMBER  { $$ = hhmm24(0, $3) }
          |             NUMBER  { $$ = hhmm24(0, $1) }
          |     minutes AFTER   { $$ = hhmm24(0, $1) }
          |     minutes TIL     { $$ = hhmm24(0, 60 - $1) }
          ;

time_in_HHMM : NUMBER ':' NUMBER              { $$ = hhmm24($1, $3) }
             | NUMBER ':' NUMBER am_or_pm     { $$ = hhmm12($1, $3, $4) }
             | NUMBER ':' NUMBER ' ' am_or_pm { $$ = hhmm12($1, $3, $5) }
             | NUMBER am_or_pm                { $$ = hhmm12($1, 0,  $2) }
             | NUMBER ' ' am_or_pm            { $$ = hhmm12($1, 0,  $3) }
             ;

weekly_spec : WEEKLY AT time_in_HHMM ON day_name { $$ = weekly($3, $5) }
            | WEEKLY    time_in_HHMM ON day_name { $$ = weekly($2, $4) }
            | WEEKLY AT time_in_HHMM    day_name { $$ = weekly($3, $4) }
            | WEEKLY    time_in_HHMM    day_name { $$ = weekly($2, $3) }
            | day_name AT time_in_HHMM           { $$ = weekly($3, $1) }
            | day_name    time_in_HHMM           { $$ = weekly($2, $1) }
            ;

am_or_pm: AM { $$ = true  }
        | PM { $$ = false }
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
             | nth_occurance_of day_name AT time_in_HHMM     { $$ = mweek($4, $2, $1) }
             | nth_occurance_of day_name    time_in_HHMM     { $$ = mweek($3, $2, $1) }
             ;

month_day : ORDINAL
          | NUMBER
          ;

nth_occurance_of : ORDINAL
                 | NUMBER
                 ;
%%
