#!/usr/bin/perl
use strict;
use warnings;

my $grep;
my %documented;
my %defined;

open $grep, "-|", q{grep Dispatch core/v2.go | sed -e 's/.*"\(.*\)".*/\1/'}
	or die "failed to grep the dispatcher code";
while (<$grep>) {
	next if m{/ui/};
	chomp;
	s{:[^/]+}{:x}g;

	# expand some regexen in the dispatch calls
	if (m/\Q(show|hide)\E/) {
		my $show = $_; $show =~ s/\Q(show|hide)\E/show/;
		my $hide = $_; $hide =~ s/\Q(show|hide)\E/hide/;
		$defined{$show} = 1;
		$defined{$hide} = 1;
	} else {
		$defined{$_} = 1;
	}

}
close $grep;

open $grep, "-|", q{grep 'name: [A-Z][A-Z]* /' docs/API.yml | sed -e 's/.*name: \(.*\) #.*/\1/'}
	or die "failed to grep API YAML docs";
while (<$grep>) {
	chomp;
	s{:[^/]+}{:x}g;
	$documented{$_} = 1;
}

my $errors = 0;
for (sort keys %defined) {
	next if $documented{$_};
	chomp;
	print "NOTDOC   $_\n";
	$errors++;
}
for (sort keys %documented) {
	next if $defined{$_};
	chomp;
	print "NOTIMPL  $_\n";
	$errors++;
}

exit 0 unless $errors;

print "$errors errors detected.\n";
exit 1
