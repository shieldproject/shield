#!/usr/bin/perl
use strict;
use warnings;

use YAML qw/Load/;
use JSON qw/decode_json/;

my $api = Load(do { local $/; <STDIN>})
	or die "$0: $!\n";

# validate
my $N = 0;
for my $section (@{$api->{sections}}) {
	$N++;
	$section->{name}
		or die "Missing section name for section $N\n";

	$section->{name} =~ s/\s+#\s+\{\{\{\s*$//;
	$section->{intro}
		or die "Missing section intro for section `$section->{name}`\n";
	$section->{endpoints}
		or die "No endpoints defined for section `$section->{name}`\n";

	my $n = 0;
	for my $endpoint (@{$section->{endpoints}}) {
		$n++;
		next if $endpoint->{FIXME};

		$endpoint->{name}
			or die "Missing endpoint name for `$section->{name}` endpoint #$n\n";

		$endpoint->{name} =~ s/\s+#\s+\{\{\{\s*$//;
		$endpoint->{name} =~ m/^(GET|PUT|POST|PATCH|DELETE|TRACE|OPTIONS)/
			or die "Unrecognized HTTP verb in `$section->{name}` endpoint `$endpoint->{name}`\n";

		if ($endpoint->{name} =~ m/^(PUT|POST|PATCH)/) {
			exists $endpoint->{request}{json}
				or die "Missing Request JSON in `$section->{name}` endpoint `$endpoint->{name}`\n";
			eval { decode_json($endpoint->{request}{json} || "{}"); 1 }
				or die "Bad Request JSON for `$section->{name}` endpoint `$endpoint->{name}`:\n!!! $@\n";
		}
		if ($endpoint->{request}{summary} && $endpoint->{request}{summary} !~ m/\{\{CURL\}\}/) {
			die "Missing {{CURL}} placeholder in Request Summary for `$section->{name}` endpoint `$endpoint->{name}`\n";
		}

		$endpoint->{response}{json}
			or die "Missing Response JSON in `$section->{name}` endpoint `$endpoint->{name}`\n";
		eval { decode_json($endpoint->{response}{json}); 1 }
			or die "Bad Response JSON for `$section->{name}` endpoint `$endpoint->{name}`:\n!!! $@\n";
		if ($endpoint->{response}{summary} && $endpoint->{response}{summary} !~ m/\{\{JSON\}\}/) {
			die "Missing {{JSON}} placeholder in Response Summary for `$section->{name}` endpoint `$endpoint->{name}`\n";
		}

		$endpoint->{errors}
			or die "Missing Errors list in `$section->{name}` endpoint `$endpoint->{name}`\n".
			       "(you can use `[]`, the empty list if no errors are possible)\n";
	}
}

# print
print $api->{intro};
for my $section (@{$api->{sections}}) {
	print "## $section->{name}\n\n";
	print "$section->{intro}\n\n";

	for my $endpoint (@{$section->{endpoints}}) {
		next if $endpoint->{FIXME};

		print "### $endpoint->{name}\n\n";
		print "$endpoint->{intro}\n\n";

		print "**Request**\n\n";
		print request($endpoint);

		print "**Response**\n\n";
		print response($endpoint);

		print "**Access Control**\n\n";
		print access($endpoint);

		if ($endpoint->{access}) {
			push @{$endpoint->{errors}}, $api->{global}{errors}{e401};
			push @{$endpoint->{errors}}, $api->{global}{errors}{e403}
				unless $endpoint->{access} eq 'any';
		}

		print "**Errors**\n\n";
		print errors($endpoint);
	}
}

sub request {
	my ($endpoint) = @_;
	my ($method, $url) = split /\s+/, $endpoint->{name};

	my $curl;

	if ($method eq "GET") {
		my $example = $endpoint->{request}{example} || $url;
		$curl .= "    curl -H 'Accept: application/json' \\\n";
		$curl .= "            https://shield.host$example\n";

	} elsif ($method =~ m/^(POST|PUT|PATCH)$/) {
		$curl .= "    curl -H 'Accept: application/json' \\\n";
		$curl .= "         -H 'Content-Type: application/json' \\\n" if $endpoint->{request}{json};
		$curl .= "         -X $method https://shield.host$url \\\n";
		if ($endpoint->{request}{json}) {
			chomp $endpoint->{request}{json};
			$curl .= "         --data-binary '\n";
			my $json = $endpoint->{request}{json};
			$json =~ s/^/    /gm;
			$curl .= "$json'\n";
		}

	} elsif ($endpoint->{name} =~ m/^DELETE /) {
		$curl .= "    curl -H 'Accept: application/json' \\\n";
		$curl .= "         -X $method https://shield.host$url \\\n";
	}

	my $qs = '';
	for my $param (@{$endpoint->{request}{query} || []}) {
		$qs .= "- **?$param->{name}=";
		$qs .= "(t|f)" if $param->{type} eq 'bool';
		$qs .= "..."   if $param->{type} eq 'string';
		$qs .= "N"     if $param->{type} eq 'number';
		$qs .= "**\n";
		$qs .= "$param->{summary}\n" if $param->{summary};
		$qs .= "\n";
	}

	$qs = "This endpoint takes no query string parameters.\n"
		unless $qs;

	my $s = $endpoint->{request}{summary} || "{{CURL}}\n\n{{QUERY}}";
	$s =~ s/\{\{CURL}}/$curl/g;
	$s =~ s/\{\{QUERY}}/$qs/g;
	return "$s\n";
}

sub response {
	my ($endpoint) = @_;
	$endpoint->{response}{json}
		or die "No response JSON specified for endpoint `$endpoint->{name}`\n";
	chomp $endpoint->{response}{json};

	my $json = "$endpoint->{response}{json}";
	$json =~ s/^/    /gm;

	my $s = $endpoint->{response}{summary} || '{{JSON}}';
	$s =~ s/\{\{JSON}}/$json/g;
	return "$s\n";
}

sub access {
	my ($endpoint) = @_;

	return "This endpoint requires no authentication or authorization.\n\n"
		unless $endpoint->{access};

	my $s;

	$s = "You must be authenticated to access this API endpoint.\n\n";
	return $s if $endpoint->{access} eq 'any';

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

	return "This API endpoint does not return any error conditions.\n\n"
		unless @{$endpoint->{errors}};

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
