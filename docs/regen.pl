#!/usr/bin/perl
use strict;
use warnings;

use YAML qw/Load/;

my $api = Load(do { local $/; <STDIN>})
	or die "$0: $!\n";

print $api->{intro};

for my $section (@{$api->{sections}}) {
	print "## $section->{name}\n\n";
	print "$section->{intro}\n\n";

	my $N = 0;
	for my $endpoint (@{$section->{endpoints}}) {
		next if $endpoint->{FIXME};
		$N++;
		die "No endpoint name given for `$section->{name}` endpoint #$N\n"
			unless $endpoint->{name};

		print "### $endpoint->{name}\n\n";
		print "$endpoint->{intro}\n\n";

		print "**Request**\n\n";
		print request($endpoint);

		print "**Response**\n\n";
		print response($endpoint);

		print "**Access Control**\n\n";
		print access($endpoint);

		print "**Errors**\n\n";
		print errors($endpoint);
	}
}

sub request {
	my ($endpoint) = @_;
	my ($method, $url) = split /\s+/, $endpoint->{name};

	my $curl;

	$curl = "```sh\n";
	if ($method eq "GET") {
		$curl .= "curl -H 'Accept: application/json' https://shield.host$url\n";

	} elsif ($method =~ m/^(POST|PUT|PATCH)$/) {
		$endpoint->{request}{json} or
			die "No request JSON specified for endpoint `$endpoint->{name}`\n";
		chomp $endpoint->{request}{json};

		$curl .= "curl -H 'Accept: application/json' \\\n";
		$curl .= "     -H 'Content-Type: application/json' \\\n";
		$curl .= "     -X $method https://shield.host$url \\\n";
		$curl .= "     --data-binary '\n$endpoint->{request}{json}'\n";

	} elsif ($endpoint->{name} =~ m/^DELETE /) {
		$curl .= "curl -H 'Accept: application/json' \\\n";
		$curl .= "     -X $method https://shield.host$url \\\n";
	}
	$curl .= "```";

	return "$curl\n\n" unless $endpoint->{request}{summary};
	my $s = $endpoint->{request}{summary};
	$s =~ s/\{\{CURL}}/$curl/g;
	return "$s\n";
}

sub response {
	my ($endpoint) = @_;
	$endpoint->{response}{json}
		or die "No response JSON specified for endpoint `$endpoint->{name}`\n";
	chomp $endpoint->{response}{json};

	my $json = "```json\n$endpoint->{response}{json}\n```";

	return "$json\n\n" unless $endpoint->{response}{summary};
	my $s = $endpoint->{response}{summary};
	$s =~ s/\{\{JSON}}/$json/g;
	return "$s\n";
}

sub access {
	my ($endpoint) = @_;

	return "This endpoint requires no authentication or authorization.\n\n"
		unless $endpoint->{access};

	my $s;

	$s = "You must be authenticated to access this API endpoint.\n\n";
	if ($endpoint->{access}[0] eq "tenant") {
		$s .= "You must also have the `$endpoint->{access}[1]` role on the tenant.\n\n";

	} elsif ($endpoint->{access}[0] eq "system") {
		$s .= "You must also have the `$endpoint->{access}[1]` system role.\n\n";

	} else {
		die "Unrecognized access type `$endpoint->{access}[0]` for `$endpoint->{name}` (must be either 'system' or 'tenant')\n";
	}

	return $s;
}

sub errors {
	my ($endpoint) = @_;

	# FIXME

	return "This API endpoint does not return any error conditions.\n\n"
		unless $endpoint->{errors} && @{$endpoint->{errors}};

	my $s;
	$s = "The following error messages can be returned:\n\n";
	for my $e (@{$endpoint->{errors}}) {
		die "No message supplied for one of the errors on `$endpoint->{name}`\n"
			unless $e->{message};

		$s .= "- **$e->{message}**";
		if ($e->{summary}) {
			my $t = $e->{summary};
			$t =~ s/^/  /smg;
			$s .= ":\n$t";
		}
		$s .= "\n";
	}

	return "$s\n";
}
