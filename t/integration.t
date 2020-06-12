#!/usr/bin/env perl
use strict;
use warnings;

use Test::More;
use Test::Deep;
use LWP::UserAgent;
use HTTP::Request;
use File::Temp qw/tempdir/;
use JSON qw/decode_json encode_json/;
use POSIX qw/strftime/;
use MIME::Base64 qw/encode_base64/;

$ENV{PUBLIC_PORT} = 9009;

my $UA = LWP::UserAgent->new(agent => 'shield-test/'.strftime("%Y%m%dT%H%M%S", gmtime())."-$$");
my $BASE_URL = "http://127.0.0.1:$ENV{PUBLIC_PORT}";

sub maybe_json {
	my ($raw) = @_;
	return eval { return decode_json($raw) } or undef;
}

diag "setting up docker-compose integration environment...\n";
system('t/setup');
is $?, 0, 't/setup should exit zero (success)'
  or do { done_testing; exit; };

my $WORKDIR = tempdir( CLEANUP => 1 );

sub run {
	my $pid = fork;
	if ($pid) {
		waitpid $pid, 0;
	} else {
		open STDOUT, ">", "$WORKDIR/stdout";
		open STDERR, ">", "$WORKDIR/stderr";
		exec @_;
		exit 126;
	}
}

sub uuid {
	return re(qr/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/),
}

my $RESULT;
sub result_is {
	my ($got, $want, $msg);
	if (ref($_[0]) && ref($_[1])) {
		($got, $want, $msg) = @_;
	} else {
		$got = $RESULT;
		($want, $msg) = @_;
	}
	$msg ||= 'should get the resulting data response we expect';
	cmp_deeply($got, $want, $msg)
		or diag explain $RESULT;
};

sub shield {
	my @args = @_;
	run('./shield', '--yes', @_);
	is($?, 0, "`shield ".join(' ', @_)."' should succeed")
		or do {
			open my $fh, "<", "$WORKDIR/stderr"
				or die "$WORKDIR/stderr: $!\n";
			my $stderr = do { local $/; <$fh>; };
			diag "failed; stderr: ----------------------------\n$stderr\n\n";
			close $fh;
		};
	open my $fh, "<", "$WORKDIR/stdout";
	$RESULT = maybe_json(do { local $/; <$fh>; });
	close $fh;
}

$ENV{SHIELD_CLI_CONFIG} = "$WORKDIR/.shield";
$ENV{SHIELD_JSON_MODE} = 'yes';
$ENV{SHIELD_BATCH_MODE} = 'yes';
$ENV{SHIELD_CORE} = 'integration-tests';
system("./shield api integration-tests $BASE_URL");
system("./shield login --username admin --password password");

shield('create-tenant', '--name' => 'Stark Enterprises');
$ENV{SHIELD_TENANT} = "Stark Enterprises";

subtest "test environment data setup" => sub { # {{{
	shield('create-user',
		'--name'        => 'Tony Stark',
		'--username'    => 'tony',
		'--password'    => 'J.A.R.V.I.S.',
		'--system-role' => 'admin');

	shield('invite',
		'--tenant' => 'Stark Enterprises',
		'--role'   => 'admin',
		'tony');

	shield('create-user',
		'--name'     => 'J.A.R.V.I.S.',
		'--username' => 'jarvis',
		'--password' => 'T.O.N.Y.');

	shield('invite',
		'--tenant' => 'Stark Enterprises',
		'--role'   => 'operator',
		'jarvis');

	shield('create-tenant', '--name' => 'Wayne Industries');

	shield('create-user',
		'--name'     => 'Bruce Wayne',
		'--username' => 'bruce',
		'--password' => 'by-day');

	shield('invite',
		'--tenant' => 'Wayne Industries',
		'--role'   => 'operator',
		'bruce');

	shield('create-user',
		'--name'     => 'Batman',
		'--username' => 'batman',
		'--password' => 'by-knight');

	shield('invite',
		'--tenant' => 'Wayne Industries',
		'--role'   => 'admin',
		'batman');

	shield('login',
		'--username' => 'tony',
		'--password' => 'J.A.R.V.I.S.');

	diag "checking that initial database is empty...";
	for (qw(jobs targets auth-tokens)) {
		shield($_); result_is([], "we should have no $_");
	}

	shield('tenants');
	result_is([map { $_->{name} } @$RESULT], bag(
		'Default Tenant',
		'Wayne Industries',
		'Stark Enterprises',
	), "we should have only the tenants we have created");

	shield('users');
	result_is([map { $_->{account} } @$RESULT], bag(qw(
		admin
		tony jarvis
		bruce batman
	)), "we should have only the users we have created");


	diag "creating testing objects...";
	shield('create-target',
		'--name'    => 'redis-shared',
		'--summary' => 'Shared Redis services for CF',
		'--agent'   => 'agent:5444',
		'--plugin'  => 'redis',
		'--data'    => 'host=127.0.0.1',
		'--data'    => 'bgsave=BGSAVE');

	shield('create-target',
		'--name'    => 'shield',
		'--summary' => 'SHIELD itself',
		'--agent'   => 'agent:5444',
		'--plugin'  => 'postgres');

	shield('create-job',
		'--name'     => 'redis-daily',
		'--summary'  => 'Daily Backups of Redis',
		'--exact',
		'--target'   => 'redis-shared',
		'--bucket'   => 'snapshots',
		'--schedule' => 'daily at 11:24pm',
		'--retain'   => '8',
		'--paused');

	shield('create-job',
		'--name'     => 'shield-itself',
		'--summary'  => 'Backing up SHIELD database, via SHIELD...',
		'--exact',
		'--target'   => 'shield',
		'--bucket'   => 'ephemeral',
		'--schedule' => 'tuesdays at 11am',
		'--retain'   => '100');
};
# }}}
subtest "auth tokens" => sub { # {{{
	shield('create-auth-token', 'test1');
	result_is({
		uuid       => uuid(),
		name       => 'test1',
		session    => re(qr/./),
		created_at => ignore(),
		last_seen  => ignore(),
	});

	shield('auth-tokens');
	result_is([map { $_->{name} } @$RESULT], [qw[test1]]);

	shield('revoke-auth-token', 'test1');
	shield('auth-tokens');
	result_is([], 'no more auth tokens');
};
# }}}
subtest "tenants" => sub { # {{{
	shield('tenant', 'Stark Enterprises');
	result_is(superhashof({
		uuid       => uuid(),
		name       => 'Stark Enterprises',
	}));

	shield('tenants');
	result_is([map { $_->{name} } @$RESULT], bag(
		'Default Tenant',
		'Stark Enterprises',
		'Wayne Industries',
	));

	shield('tenants', 'Stark');
	result_is([map { $_->{name} } @$RESULT], bag(
		'Stark Enterprises',
	), 'partial tenant name search should work');

	shield('create-tenant', '--name', 'My New Tenant');
	shield('tenant', 'My New Tenant');
	result_is(superhashof({
		uuid => uuid(),
		name => 'My New Tenant'
	}));

	shield('update-tenant', 'My New Tenant',
		'--name' => 'My Updated Tenant');
	shield('tenant', 'My Updated Tenant');
	result_is(superhashof({
		uuid => uuid(),
		name => 'My Updated Tenant',
		members => undef,
	}));

	shield('invite',
		'--tenant' => 'My Updated Tenant',
		'--role'   => 'operator',
		'tony', 'jarvis');

	shield('tenant', 'My Updated Tenant');
	result_is([map { "$_->{account} / $_->{role}" } @{ $RESULT->{members} }],
		[ 'tony / operator',
		  'jarvis / operator' ]);

	shield('invite',
		'--tenant' => 'My Updated Tenant',
		'--role'   => 'engineer',
		'jarvis');

	shield('tenant', 'My Updated Tenant');
	result_is([map { "$_->{account} / $_->{role}" } @{ $RESULT->{members} }],
		[ 'tony / operator',
		  'jarvis / engineer' ]);

	shield('banish',
		'--tenant' => 'My Updated Tenant',
		'tony');

	shield('tenant', 'My Updated Tenant');
	result_is([map { "$_->{account} / $_->{role}" } @{ $RESULT->{members} }],
		[ 'jarvis / engineer' ]);

	# banish tony again, for good measure
	shield('banish',
		'--tenant' => 'My Updated Tenant',
		'tony');

	shield('tenant', 'My Updated Tenant');
	result_is([map { "$_->{account} / $_->{role}" } @{ $RESULT->{members} }],
		[ 'jarvis / engineer' ]);
};
# }}}
subtest "users" => sub { # {{{
	shield('user', 'jarvis');
	result_is(superhashof({
		uuid       => uuid(),
		name       => 'J.A.R.V.I.S.',
		account    => 'jarvis',
		sysrole    => '',
	}));

	shield('user', 'tony');
	result_is(superhashof({
		uuid       => uuid(),
		name       => 'Tony Stark',
		account    => 'tony',
		sysrole    => 'admin',
	}));

	shield('users');
	result_is([map { $_->{account} } @$RESULT], bag(qw/
		admin
		tony jarvis
		bruce batman
	/));
	result_is([grep { $_->{account} eq 'tony' } @$RESULT], [superhashof({
		uuid    => uuid(),
		name    => 'Tony Stark',
		account => 'tony',
		sysrole => 'admin',
	})]);

	shield('users', '--fuzzy', 'a');
	result_is([map { $_->{account} } @$RESULT], bag(qw/
		admin
		jarvis
		batman
	/));

	shield('users', '--fuzzy', 'xyzzy');
	result_is([]);

	shield('users', '--with-system-role' => 'admin');
	result_is([map { $_->{account} } @$RESULT], bag(qw/
		admin
		tony
	/));

	shield('create-user',
		'--name'     => 'Some User',
		'--username' => 'user42',
		'--password' => 'temp-password');
	shield('user', 'user42');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'Some User',
		account => 'user42',
		sysrole => '',
	}));

	shield('update-user', 'user42',
		'--system-role' => 'engineer');
	shield('user', 'user42');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'Some User',
		account => 'user42',
		sysrole => 'engineer',
	}));

	shield('update-user', 'user42',
		'--name' => 'Some Other User');
	shield('user', 'user42');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'Some Other User',
		account => 'user42',
		sysrole => 'engineer',
	}));
};
# }}}
subtest "targets" => sub { # {{{
	shield('target', 'redis-shared');
	result_is(superhashof({
		uuid       => uuid(),
		name       => 'redis-shared',
		summary    => 'Shared Redis services for CF',
		agent      => 'agent:5444',
		plugin     => 'redis',
		config     => {
		                host   => '127.0.0.1',
		                bgsave => 'BGSAVE',
		              },
	}));

	shield('targets');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		redis-shared
		shield
	]));

	shield('targets', 'redis-shared');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		redis-shared
	]));

	shield('targets', '--fuzzy', 'redis');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		redis-shared
	]));

	shield('targets', 's');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		shield
		redis-shared
	]));

	shield('targets', '--used');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		shield
		redis-shared
	]));

	shield('targets', '--unused');
	result_is([]); # FIXME

	shield('targets', '--with-plugin', 'redis');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		redis-shared
	]));

	shield('targets', '--with-plugin', 'enoent');
	result_is([]);

	shield('targets', '--with-plugin', 'redis', '--used');
	result_is([map { $_->{name} } @$RESULT], bag(qw[
		redis-shared
	]));

	shield('targets', '--with-plugin', 'redis', '--unused');
	result_is([]);
};
# }}}
subtest "target lifecycle" => sub { # {{{
	shield('create-target',
		'--name'    => 'My New Target',
		'--summary' => 'A Target for editing',
		'--agent'   => 'agent:5444',
		'--plugin'  => 'fs',
		'--data'    => 'dir=/path/to/data');

	shield('target', 'My New Target');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'My New Target',
		summary => 'A Target for editing',
		agent   => 'agent:5444',
		plugin  => 'fs',
		config  => {
		             dir => '/path/to/data',
		           },
	}));

	shield('update-target', 'My New Target',
		'--name'    => 'My Updated Target',
		'--summary' => 'New Summary',
		'--data'    => 'dir=/new/path');

	shield('target', 'My Updated Target');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'My Updated Target',
		summary => 'New Summary',
		agent   => 'agent:5444',
		plugin  => 'fs',
		config  => {
		             dir => '/new/path',
		           },
	}));

	shield('update-target', 'My Updated Target',
		'--data' => 'new=data');

	shield('target', 'My Updated Target');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'My Updated Target',
		summary => 'New Summary',
		agent   => 'agent:5444',
		plugin  => 'fs',
		config  => {
		             dir => '/new/path',
		             new => 'data',
		           },
	}));

	shield('update-target', 'My Updated Target',
		'--clear-data',
		'--data' => 'dir=/another/path');

	shield('target', 'My Updated Target');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'My Updated Target',
		summary => 'New Summary',
		agent   => 'agent:5444',
		plugin  => 'fs',
		config  => {
		             dir => '/another/path',
		           },
	}));

	shield('update-target', 'My Updated Target',
		'--plugin' => 'postgres');

	shield('target', 'My Updated Target');
	result_is(superhashof({
		uuid    => uuid(),
		name    => 'My Updated Target',
		summary => 'New Summary',
		agent   => 'agent:5444',
		plugin  => 'postgres',
		config  => undef,
	}));

	# FIXME check that we cannot delete redis-shared
	# FIXME check that we can delete 'My Updated Target'
};
# }}}
subtest "jobs" => sub { # {{{
	shield('job', 'redis-daily');
	result_is(superhashof({
		uuid      => uuid(),
		name      => 'redis-daily',
		summary   => 'Daily Backups of Redis',
		schedule  => 'daily at 11:24pm',
		keep_days => 8,
		bucket    => ignore(),
		paused    => bool(1),
		target    => superhashof({
		               uuid     => uuid(),
		               name     => 'redis-shared',
		               plugin   => 'redis',
		               endpoint => '{"bgsave":"BGSAVE","host":"127.0.0.1"}',
		             }),
	}));

	shield('jobs');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		redis-daily
		shield-itself
	)));

	shield('jobs', 'redis-daily');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		redis-daily
	)));

	shield('jobs', '--fuzzy', 'daily');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		redis-daily
	)));

	shield('jobs', '--fuzzy', 'e');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		redis-daily
		shield-itself
	)));

	shield('jobs', '--fuzzy', 'xyzz');
	result_is([]);

	shield('target', 'shield');
	shield('jobs', '--target', $RESULT->{uuid});
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		shield-itself
	)));

	shield('jobs', '--bucket', 'ephemeral');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		shield-itself
	)));

	shield('jobs', '--paused');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		redis-daily
	)));

	shield('jobs', '--unpaused');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		shield-itself
	)));

	shield('pause-job', 'shield-itself');
	shield('unpause-job', 'redis-daily');
	shield('jobs', '--paused');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		shield-itself
	)));
	shield('jobs', '--unpaused');
	result_is([map { $_->{name} } @$RESULT], bag(qw(
		redis-daily
	)));


	shield('create-job',
		'--name'     => 'New Job',
		'--summary'  => 'edit me!',
		'--schedule' => 'daily 4am',
		'--target'   => 'shield',
		'--bucket'   => 'snapshots',
		'--retain'   => '5');
	shield('job', 'New Job');
	result_is(superhashof({
		uuid      => uuid(),
		summary   => 'edit me!',
		schedule  => 'daily 4am',
		keep_days => 5,
		target    => superhashof({ name => 'shield' }),
		bucket    => 'snapshots',
	}));

	shield('update-job', 'New Job',
		'--name'    => 'Updated Job',
		'--summary' => 'New Summary');
	shield('job', 'Updated Job');
	result_is(superhashof({
		uuid      => uuid(),
		summary   => 'New Summary',
		schedule  => 'daily 4am',
		keep_days => 5,
		target    => superhashof({ name => 'shield' }),
		bucket    => 'snapshots',
	}));

	shield('update-job', 'Updated Job',
		'--schedule' => 'daily 3:30am',
		'--retain'   => '8');
	shield('job', 'Updated Job');
	result_is(superhashof({
		uuid      => uuid(),
		summary   => 'New Summary',
		schedule  => 'daily 3:30am',
		keep_days => 8,
		target    => superhashof({ name => 'shield' }),
		bucket    => 'snapshots',
	}));
};
# }}}


done_testing;
