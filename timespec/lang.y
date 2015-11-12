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
