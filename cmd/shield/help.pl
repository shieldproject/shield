#!/usr/bin/perl
use strict;
use warnings;

my $CODE = do { local $/; <DATA> };
my $CASES = "";
for my $file (@ARGV) {
	open my $fh, "<", $file
		or die "Failed to open $file for reading: $!\n";

	my $help = do { local $/; <$fh> };
	close $fh;

	$help =~ s/\\/\\\\/g;
	$help =~ s/"/\\"/g;
	$help = join("", map { "\t\tfmt.Printf(\"$_\\n\")\n" } split("\n", $help, -1));

	my $command = $file; $command =~ s|.*/||; $command =~ s/--/ /;
	$CASES .= "\tcase \"$command\": /* {{{ */\n$help\n\t/* }}} */\n";
}
$CODE =~ s/__CASES__/$CASES/;
print $CODE;

__DATA__
package main

import (
	fmt "github.com/jhunt/go-ansi"
)

func ShowHelp(command string) {
	switch command {
__CASES__
	default:
		fmt.Printf("No help is available for `@G{shield} @R{%s}` yet.\n\n", command)
	}
}
