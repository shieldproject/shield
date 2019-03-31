QUnit.assert.empty = function (lst, message) {
  this.pushResult({
    result:   (lst instanceof Array) && lst.length == 0,
    actual:   lst,
    expected: [],
    message:  message || 'list should be empty'
  });
};

var contains = function (got, want) {
  for (var key in want) {
    if (want.hasOwnProperty(key) && got[key] != want[key]) {
      return false;
    }
  }
  return true;
};

QUnit.assert.contained = function (actual, expected, message) {
  this.pushResult({
    result:   actual ? contains(actual, expected) : false,
    actual:   actual,
    expected: expected,
    message:  message || 'object mismatch'
  });
};

QUnit.assert.set = function (actual, expected, message) {
  var result = true;
  if (!(actual instanceof Array)
   || actual.length != expected.length) {
    result = false;
  } else {
    for (var i = 0; i < actual.length; i++) {
      var ok = false;
      for (var j = 0; j < expected.length; j++) {
        if (expected[j] && contains(actual[i], expected[j])) {
          expected[j] = null;
          ok = true;
          break;
        }
      }
      if (!ok) {
        result = false;
        break;
      }
    }
  }

  this.pushResult({
    result:   result,
    actual:   actual,
    expected: expected,
    message:  message || 'should be a proper set congruent to expected'
  });
};

QUnit.assert.any = function (actual, expected, message) {
  for (var i = 0; i < actual.length; i++) {
    if (contains(actual[i], expected)) {
      this.pushResult({
        result:   true,
        actual:   actual,
        expected: expected,
        message:  message || 'object should be contained in the list'
      });
      return
    }
  }
  this.pushResult({
    result:   false,
    actual:   actual,
    expected: expected,
    message:  message || 'object should be contained in the list'
  });
};

QUnit.module('AEGIS Data Operations');
QUnit.test('Basic Operations', function(is) {
  var thing, db = $.aegis();
  is.ok(typeof(db) !== 'undefined',
        '$.aegis() returns a valid object');

  thing = db.find('thing', { uuid: 'thing-the-first' });
  is.ok(typeof(thing) === 'undefined',
        'a blank AEGIS database contains no data');

  db.insert('thing', { uuid: 'thing-the-first', name: 'a thing' });
  thing = db.find('thing', { uuid: 'thing-the-first' });
  is.ok(typeof(thing) !== 'undefined',
        'after inserting a thing, find() returns it.');
  is.equal(thing.uuid, 'thing-the-first',
           'find() found the thing with the correct UUID');
  is.equal(thing.name, 'a thing',
           'find() found the thing with the correct UUID');

  thing = db.find('thing', { uuid: 'not-a-real-thing' });
  is.ok(typeof(thing) === 'undefined',
        'find() still returns undefined for nonexistent things');

  db.update('thing', { uuid: 'thing-the-first', name: 'updated name' });
  thing = db.find('thing', { uuid: 'thing-the-first' });
  is.ok(typeof(thing) !== 'undefined',
        'after updating a thing, find() returns it.');
  is.equal(thing.uuid, 'thing-the-first',
           'update() does not change the UUID');
  is.equal(thing.name, 'updated name',
           'update() changes the name (as instructed)');

  db.delete('thing', { uuid: 'thing-the-first' });
  thing = db.find('thing', { uuid: 'thing-the-first' });
  is.ok(typeof(thing) === 'undefined',
        'objects can be deleted from the database');
});

QUnit.test('Insertion', function (is) {
  var thing, db = $.aegis();

  is.ok(!db.insert('thing', { no_uuid: true }),
        'Inserting a thing without a UUID should fail');

  db.insert('thing', { uuid: '1-2-3', key1: 'value1' });
  thing = db.find('thing', { uuid: '1-2-3' });
  is.ok(typeof(thing) !== 'undefined', 'thing 1-2-3 exists after insert');

  is.equal(thing.key1, 'value1',
           'insert() keeps the attributes that were inserted.');
  is.ok(!('key2' in thing),
        'key2 (which was not in the insert object) does not exist.');

  db.insert('thing', { uuid: '1-2-3', key2: 'value2' });
  thing = db.find('thing', { uuid: '1-2-3' });
  is.ok(typeof(thing) !== 'undefined', 'thing 1-2-3 exists after insert');

  is.equal(thing.key2, 'value2',
           'insert() keeps the attributes that were inserted.');
  is.ok(!('key1' in thing),
        'key1 no longer exists (insert is an overwrite)');
});

QUnit.test('Updates', function (is) {
  var thing, db = $.aegis();

  db.insert('thing', { uuid: '1-2-3', key1: 'value1' });
  thing = db.find('thing', { uuid: '1-2-3' });
  is.ok(typeof(thing) !== 'undefined', 'thing 1-2-3 exists after insert');

  is.equal(thing.key1, 'value1',
           'insert() keeps the attributes that were inserted.');
  is.ok(!('key2' in thing),
        'key2 (which was not in the insert object) does not exist.');

  db.update('thing', { uuid: '1-2-3', key2: 'value2' });
  thing = db.find('thing', { uuid: '1-2-3' });
  is.ok(typeof(thing) !== 'undefined', 'thing 1-2-3 exists after update');

  is.equal(thing.key1, 'value1',
           'update() keeps the attributes that were originally inserted.');
  is.equal(thing.key2, 'value2',
           'update() adds the attributes that were in the update.');
});

QUnit.module('AEGIS Object Queries');
(function () {
  var Dataset = function () {
    return $.aegis()

      /* TENANTS */
      .insert('tenant', {
        uuid: 'the-system-tenant',
        name: 'tenant1'
      })
      .insert('tenant', {
        uuid: 'the-acme-tenant',
        name: 'Acme, Inc'
      })

      /* TARGETS */
      .insert('target', {
        uuid: 'the-shield-target',
        name: 'The SHIELD Target',
        tenant_uuid: 'the-system-tenant'
      })
      .insert('target', {
        uuid: 'the-ccdb-target',
        name: 'The CCDB Target',
        tenant_uuid: 'the-acme-tenant'
      })
      .insert('target', {
        uuid: 'the-uaadb-target',
        name: 'The UAADB Target',
        tenant_uuid: 'the-acme-tenant'
      })

      /* STORES */
      .insert('store', {
        uuid: 'the-global-store',
        name: 'The Global Store',
        global: true
      })
      .insert('store', {
        uuid: 'the-system-s3-store',
        name: 'The System S3 Store',
        tenant_uuid: 'the-system-tenant'
      })

      /* JOBS */
      .insert('job', {
        uuid: 'the-shield-daily-job',
        name: 'Daily',
        schedule: 'daily 3:35am',
        tenant_uuid: 'the-system-tenant',
        store_uuid:  'the-system-s3-store',
        target_uuid: 'the-shield-target'
      })
      .insert('job', {
        uuid: 'the-shield-weekly-job',
        name: 'Weekly',
        schedule: 'weekly on sundays at 6:15am',
        tenant_uuid: 'the-system-tenant',
        store_uuid:  'the-global-store',
        target_uuid: 'the-shield-target'
      })
      .insert('job', {
        uuid: 'the-ccdb-hourly-job',
        name: 'Hourly',
        schedule: 'every hour at *:25',
        tenant_uuid: 'the-acme-tenant',
        store_uuid:  'the-global-store',
        target_uuid: 'the-ccdb-target'
      })
      .insert('job', {
        uuid: 'the-uaadb-hourly-job',
        name: 'Hourly',
        schedule: 'every hour at *:25',
        tenant_uuid: 'the-acme-tenant',
        store_uuid:  'the-global-store',
        target_uuid: 'the-uaadb-target'
      })

      /* TASKS */
      .insert('task', {
        uuid:   'shield-backup-task-1',
        op:     'backup',
        status: 'done',
        tenant_uuid:  'the-system-tenant',
        job_uuid:     'the-shield-daily-job',
        archive_uuid: 'shield-backup-archive-1',
        store_uuid:   'the-system-s3-store',
        target_uuid:  'the-shield-target'
      })
      .insert('task', {
        uuid:   'shield-backup-task-2',
        op:     'backup',
        status: 'done',
        tenant_uuid:  'the-system-tenant',
        job_uuid:     'the-shield-daily-job',
        archive_uuid: 'shield-backup-archive-2',
        store_uuid:   'the-system-s3-store',
        target_uuid:  'the-shield-target'
      })
      .insert('task', {
        uuid:   'shield-purge-task-1',
        op:     'purge',
        status: 'done',
        tenant_uuid:  'the-system-tenant',
        archive_uuid: 'shield-backup-archive-1',
        store_uuid:   'the-system-s3-store'
      })
      .insert('task', {
        uuid:   'shield-backup-task-3',
        op:     'backup',
        status: 'done',
        tenant_uuid:  'the-system-tenant',
        job_uuid:     'the-shield-weekly-job',
        archive_uuid: 'shield-backup-archive-3',
        store_uuid:   'the-global-store',
        target_uuid:  'the-shield-target'
      })

      .insert('task', {
        uuid:   'ccdb-backup-task-1',
        op:     'backup',
        status: 'done',
        tenant_uuid:  'the-acme-tenant',
        job_uuid:     'the-ccdb-hourly-job',
        archive_uuid: 'ccdb-backup-archive-1',
        store_uuid:   'the-global-store',
        target_uuid:  'the-ccdb-target'
      })

      .insert('task', {
        uuid:   'uaadb-backup-task-1',
        op:     'backup',
        status: 'done',
        tenant_uuid:  'the-acme-tenant',
        job_uuid:     'the-uaadb-hourly-job',
        archive_uuid: 'uaadb-backup-archive-1',
        store_uuid:   'the-global-store',
        target_uuid:  'the-uaadb-target'
      })

      .insert('task', {
        uuid:   'test-global-store-task-1',
        op:     'test-store',
        status: 'done',
        store_uuid:  'the-global-store'
      })
      .insert('task', {
        uuid:   'test-system-s3-store-task-1',
        op:     'test-store',
        status: 'done',
        tenant_uuid: 'the-system-tenant',
        store_uuid:  'the-system-s3-store'
      })

      /* ARCHIVES */
      .insert('archive', {
        uuid: 'shield-backup-archive-1',
        tenant_uuid: 'the-system-tenant',
        target_uuid: 'the-shield-target',
        store_uuid:  'the-system-s3-store'
      })
      .insert('archive', {
        uuid: 'shield-backup-archive-2',
        tenant_uuid: 'the-system-tenant',
        target_uuid: 'the-shield-target',
        store_uuid:  'the-system-s3-store'
      })
      .insert('archive', {
        uuid: 'shield-backup-archive-3',
        tenant_uuid: 'the-system-tenant',
        target_uuid: 'the-shield-target',
        store_uuid:  'the-global-store'
      })
      .insert('archive', {
        uuid: 'ccdb-backup-archive-1',
        tenant_uuid: 'the-acme-tenant',
        target_uuid: 'the-ccdb-target',
        store_uuid:  'the-global-store'
      })
      .insert('archive', {
        uuid: 'uaadb-backup-archive-1',
        tenant_uuid: 'the-acme-tenant',
        target_uuid: 'the-uaadb-target',
        store_uuid:  'the-global-store'
      })
    ;
  };

  QUnit.test('Tenant Retrieval', function (is) {
    var db = Dataset();

    is.set(db.tenants(),
      [ { uuid: 'the-system-tenant' },
        { uuid: 'the-acme-tenant' } ],
      'without a filter, all tenants should be retrieved');

    is.contained(db.tenant('the-acme-tenant'),
      { name: 'Acme, Inc' },
      'the-acme-tenant can be retrieved directly');
    is.ok(!db.tenant('a-nonexistent-tenant'),
          'a non-existent-tenant cannot be retrieved');
  });

  QUnit.test('System Retrieval', function (is) {
    var db = Dataset();

    is.set(db.systems({ tenant: 'the-system-tenant' }),
      [ { name: 'The SHIELD Target' } ],
      'the-system-tenants single system is SHIELD');

    is.set(db.systems({ tenant: 'the-acme-tenant' }),
      [ { name: 'The CCDB Target' },
        { name: 'The UAADB Target' } ],
      'the-acme-tenant has two systems: CCDB and UAADB');

    is.empty(db.systems({ tenant: 'a-non-existent-tenant' }),
      'a-non-existent-tenant has no targets/systems');

    /* single-system retrieval */
    is.contained(
      db.system('the-ccdb-target'),
      { name: 'The CCDB Target' },
      'the-ccdb-target system exists and can be retrieved');
    is.contained(
      db.system('the-uaadb-target'),
      { name: 'The UAADB Target' },
      'the-uaadb-target system exists and can be retrieved');
    is.ok(!db.system('a-nonexistent-target'),
          'a-nonexistent-target system cannot be retrieved');
  });

  QUnit.test('Store Retrieval', function (is) {
    var db = Dataset();

    is.set(db.stores(),
      [ { name: 'The Global Store' },
        { name: 'The System S3 Store' } ],
      'without a filter, all stores should be retrieved');

    is.set(db.stores({ tenant: 'the-system-tenant' }),
      [ { name: 'The System S3 Store' } ],
      'the-system-tenants single store is System S3');

    is.empty(db.stores({ tenant: 'the-acme-tenant' }),
      'the-acme-tenant should have no stores');

    is.set(db.stores({ tenant: 'the-system-tenant',
                       global: true }),
      [ { name: 'The System S3 Store' },
        { name: 'The Global Store' } ],
      'the-system-tenant should have access to global and tenant-specific stores');

    is.empty(db.stores({ tenant: 'a-non-existent-tenant' }),
      'a-non-existent-tenant has no stores');

    is.contained(
      db.store('the-system-s3-store'),
      { name: 'The System S3 Store' },
      'the-system-s3-store store exists and can be retrieved');
    is.contained(
      db.store('the-global-store'),
      { name: 'The Global Store' },
      'the-global-store store exists and can be retrieved');
    is.ok(!db.store('a-nonexistent-store'),
          'a-nonexistent-store cannot be retrieved');

    /* single store retrieval */
    is.contained(
      db.store('the-system-s3-store'),
      { name: 'The System S3 Store' },
      'the-system-s3-store can be retrieved');
    is.ok(!db.store('a-nonexistent-store'),
          'a non-existent-store cannot be retrieved');
  });

  QUnit.test('Job Retrieval', function (is) {
    var db = Dataset();

    is.set(db.jobs({ system: 'the-shield-target' }),
      [ { name: 'Daily',  schedule: 'daily 3:35am' },
        { name: 'Weekly', schedule: 'weekly on sundays at 6:15am' } ],
      'the-shield-target has a daily job (3:35am) and a weekly job (sun 6:15am)');

    is.set(db.jobs({ system: 'the-shield-target',
                     tenant: 'the-system-tenant' }),
      [ { name: 'Daily',  schedule: 'daily 3:35am' },
        { name: 'Weekly', schedule: 'weekly on sundays at 6:15am' } ],
      'the-shield-target (on the system-tenant) has a daily job (3:35am) and a weekly job (sun 6:15am)');

    is.empty(db.jobs({ system: 'the-shield-target',
                       tenant: 'the-acme-tenant' }),
      'the-acme-tenant does not have the-shield-target as a system, so it can have no jobs for it');

    /* single job retrieval */
    is.contained(
      db.job('the-shield-daily-job'),
      { tenant_uuid: 'the-system-tenant',
        target_uuid: 'the-shield-target',
        store_uuid:  'the-system-s3-store' },
      'the-shield-daily-job can be retrieved');
    is.ok(!db.job('a-nonexistent-job'),
          'a non-existent-job cannot be retrieved');
  });

  QUnit.test('Task Retrieval', function (is) {
    var db = Dataset();

    is.set(db.tasks({ tenant: 'the-system-tenant' }),
      [ { uuid: 'test-system-s3-store-task-1' },
        { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-backup-task-2' },
        { uuid: 'shield-backup-task-3' },
        { uuid: 'shield-purge-task-1'  } ],
      'the-system-tenant has four total tasks');

    is.set(db.tasks({ tenant: 'the-system-tenant',
                      system: 'the-shield-target' }),
      [ { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-backup-task-2' },
        { uuid: 'shield-backup-task-3' } ],
      'the-system-tenant has three total tasks for system the-shield-target');

    is.set(db.tasks({ tenant: 'the-system-tenant',
                      system: 'the-shield-target',
                      job:    'the-shield-weekly-job' }),
      [ { uuid: 'shield-backup-task-3' } ],
      'the-system-tenant has one task for system the-shield-target (weekly job)');

    is.set(db.tasks({ tenant: 'the-system-tenant',
                      store:  'the-system-s3-store' }),
      [ { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-backup-task-2' },
        { uuid: 'shield-purge-task-1'  },
        { uuid: 'test-system-s3-store-task-1' } ],
      'the-system-tenant has three tasks for system the-shield-target in the-system-s3-store');

    is.set(db.tasks({ tenant: 'the-system-tenant',
                      archive: 'shield-backup-archive-1' }),
      [ { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-purge-task-1'  } ],
      'the-system-tenant has two tasks for system the-shield-target, archive shield-backup-archive-1');

    /* single task retrieval */
    is.contained(
      db.task('shield-backup-task-1'),
      { tenant_uuid: 'the-system-tenant',
        target_uuid: 'the-shield-target',
        store_uuid:  'the-system-s3-store' },
      'the shield-backup-task-1 task can be retrieved');
    is.ok(!db.task('a-nonexistent-task'),
          'a non-existent-task task cannot be retrieved');
  });

  QUnit.test('Archive Retrieval', function (is) {
    var db = Dataset();

    /* by tenant */
    is.set(db.archives({tenant: 'the-system-tenant' }),
      [ { uuid: 'shield-backup-archive-1' },
        { uuid: 'shield-backup-archive-2' },
        { uuid: 'shield-backup-archive-3' } ],
      'the-system-tenant has three archives total');

    /* by tenant + target */
    is.set(db.archives({ tenant: 'the-acme-tenant',
                         system: 'the-ccdb-target' }),
      [ { uuid: 'ccdb-backup-archive-1' } ],
      'the-acme-tenant has one archive for the-ccdb-target, total');

    /* by tenant + target + store */
    is.set(db.archives({ tenant: 'the-system-tenant',
                         system: 'the-shield-target',
                         store:  'the-system-s3-store'}),
      [ { uuid: 'shield-backup-archive-1' },
        { uuid: 'shield-backup-archive-2' } ],
      'the-system-tenant has two archives for the-shield-target in the-system-s3-store');

    /* single archive retrieval */
    is.contained(
      db.archive('shield-backup-archive-1'),
      { tenant_uuid: 'the-system-tenant',
        target_uuid: 'the-shield-target',
        store_uuid:  'the-system-s3-store' },
      'the shield-backup-archive-1 archive can be retrieved');
    is.ok(!db.archive('a-nonexistent-archive'),
          'a non-existent-archive archive cannot be retrieved');
  });
})();

QUnit.module('AEGIS RBAC');
(function () {
  var Dataset = function () {
    return $.aegis()

      /* TENANTS */
      .insert('tenant', {
        uuid: 'the-system-tenant',
        name: 'tenant1'
      })
      .insert('tenant', {
        uuid: 'the-acme-tenant',
        name: 'Acme, Inc'
      })
    ;
  };

  QUnit.test('System-wide Roles', function (is) {
    var AEGIS = Dataset()

    AEGIS.grant('admin');
    is.equal(AEGIS.role(), 'Administrator');
    is.ok(AEGIS.is('admin'),    'A SHIELD administrator is considered an admin');
    is.ok(AEGIS.is('manager'),  'A SHIELD administrator is considered a manager');
    is.ok(AEGIS.is('engineer'), 'A SHIELD administrator is considered an engineer');

    is.ok(!AEGIS.is('the-acme-tenant', 'manager'),  'A SHIELD administrator is NOT implicitly a tenant manager');
    is.ok(!AEGIS.is('the-acme-tenant', 'engineer'), 'A SHIELD administrator is NOT implicitly a tenant engineer');
    is.ok(!AEGIS.is('the-acme-tenant', 'operator'), 'A SHIELD administrator is NOT implicitly a tenant operator');

    AEGIS.grant('manager');
    is.equal(AEGIS.role(), 'Manager');
    is.ok(!AEGIS.is('admin'),    'A SHIELD manager is NOT considered an admin');
    is.ok( AEGIS.is('manager'),  'A SHIELD manager is considered a manager');
    is.ok( AEGIS.is('engineer'), 'A SHIELD manager is considered an engineer');

    is.ok(!AEGIS.is('the-acme-tenant', 'manager'),  'A SHIELD manager is NOT implicitly a tenant manager');
    is.ok(!AEGIS.is('the-acme-tenant', 'engineer'), 'A SHIELD manager is NOT implicitly a tenant engineer');
    is.ok(!AEGIS.is('the-acme-tenant', 'operator'), 'A SHIELD manager is NOT implicitly a tenant operator');

    AEGIS.grant('engineer');
    is.equal(AEGIS.role(), 'Engineer');
    is.ok(!AEGIS.is('admin'),    'A SHIELD engineer is NOT considered an admin');
    is.ok(!AEGIS.is('manager'),  'A SHIELD engineer is NOT considered a manager');
    is.ok( AEGIS.is('engineer'), 'A SHIELD engineer is considered an engineer');

    is.ok(!AEGIS.is('the-acme-tenant', 'manager'),  'A SHIELD engineer is NOT implicitly a tenant manager');
    is.ok(!AEGIS.is('the-acme-tenant', 'engineer'), 'A SHIELD engineer is NOT implicitly a tenant engineer');
    is.ok(!AEGIS.is('the-acme-tenant', 'operator'), 'A SHIELD engineer is NOT implicitly a tenant operator');

    AEGIS.grant('none');
    is.equal(AEGIS.role(), '');
    is.ok(!AEGIS.is('admin'),    'A SHIELD (nothing) is NOT considered an admin');
    is.ok(!AEGIS.is('manager'),  'A SHIELD (nothing) is NOT considered a manager');
    is.ok(!AEGIS.is('engineer'), 'A SHIELD (nothing) is NOT considered an engineer');

    is.ok(!AEGIS.is('the-acme-tenant', 'manager'),  'A SHIELD (nothing) is NOT implicitly a tenant manager');
    is.ok(!AEGIS.is('the-acme-tenant', 'engineer'), 'A SHIELD (nothing) is NOT implicitly a tenant engineer');
    is.ok(!AEGIS.is('the-acme-tenant', 'operator'), 'A SHIELD (nothing) is NOT implicitly a tenant operator');
  });

  QUnit.test('Tenant-Specific Roles', function (is) {
    var AEGIS = Dataset()

    AEGIS.grant('the-acme-tenant', 'admin');
    is.equal(AEGIS.role('the-acme-tenant'), 'Administrator');
    is.ok(!AEGIS.is('admin'),    'A Tenant admin is NOT considered a SHIELD admin');
    is.ok(!AEGIS.is('manager'),  'A Tenant admin is NOT considered a SHIELD manager');
    is.ok(!AEGIS.is('engineer'), 'A Tenant admin is NOT considered a SHIELD engineer');

    is.ok(AEGIS.is('the-acme-tenant', 'admin'),    'A Tenant admin is a tenant admin');
    is.ok(AEGIS.is('the-acme-tenant', 'engineer'), 'A Tenant admin is a tenant engineer');
    is.ok(AEGIS.is('the-acme-tenant', 'operator'), 'A Tenant admin is a tenant operator');

    AEGIS.grant('the-acme-tenant', 'engineer');
    is.equal(AEGIS.role('the-acme-tenant'), 'Engineer');
    is.ok(!AEGIS.is('admin'),    'A Tenant engineer is NOT considered a SHIELD admin');
    is.ok(!AEGIS.is('manager'),  'A Tenant engineer is NOT considered a SHIELD manager');
    is.ok(!AEGIS.is('engineer'), 'A Tenant engineer is NOT considered a SHIELD engineer');

    is.ok(!AEGIS.is('the-acme-tenant', 'admin'),    'A Tenant engineer is NOT a tenant admin');
    is.ok( AEGIS.is('the-acme-tenant', 'engineer'), 'A Tenant engineer is a tenant engineer');
    is.ok( AEGIS.is('the-acme-tenant', 'operator'), 'A Tenant engineer is a tenant operator');

    AEGIS.grant('the-acme-tenant', 'operator');
    is.equal(AEGIS.role('the-acme-tenant'), 'Operator');
    is.ok(!AEGIS.is('admin'),    'A Tenant operator is NOT considered a SHIELD admin');
    is.ok(!AEGIS.is('manager'),  'A Tenant operator is NOT considered a SHIELD manager');
    is.ok(!AEGIS.is('engineer'), 'A Tenant operator is NOT considered a SHIELD engineer');

    is.ok(!AEGIS.is('the-acme-tenant', 'admin'),    'A Tenant operator is NOT a tenant admin');
    is.ok(!AEGIS.is('the-acme-tenant', 'engineer'), 'A Tenant operator is NOT a tenant engineer');
    is.ok( AEGIS.is('the-acme-tenant', 'operator'), 'A Tenant operator is a tenant operator');
  });

  QUnit.test('Default Tenant Selection', function (is) {
    var AEGIS = Dataset().grant('the-acme-tenant', 'admin');

    is.ok( AEGIS.is('the-acme-tenant',   'admin'), '[sanity check] should be an admin on acme');
    is.ok(!AEGIS.is('the-system-tenant', 'admin'), '[sanity check] should not be an admin on systme');

    AEGIS.use('the-acme-tenant');
    is.ok(AEGIS.is('tenant', 'admin'), 'AEGIS should use selected tenant for UUID "tenant"');
    AEGIS.use('the-system-tenant');
    is.ok(!AEGIS.is('tenant', 'admin'), 'When the-system-tenant is current, we are not an admin');
  });
})();
