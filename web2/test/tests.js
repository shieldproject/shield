QUnit.assert.empty = function (lst, message) {
  console.log(lst, lst.length);
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
    result:   contains(actual, expected),
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

QUnit.module('SHIELD AEGIS');
QUnit.test('Basic Operations', function(is) {
  var thing, db = new AEGIS();
  is.ok(typeof(db) !== 'undefined',
        'new AEGIS() returns a valid object');

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
  var thing, db = new AEGIS();

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
  var thing, db = new AEGIS();

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

{
  var Dataset = function () {
    return new AEGIS()

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

  QUnit.test('System Retrieval', function (is) {
    var system, systems, db = Dataset();

    systems = db.systems({ tenant: 'the-system-tenant' })
    is.ok(systems, 'the-system-tenant has targets/systems');
    is.equal(systems.length, 1, 'the-system-tenant has a single system');
    is.equal(systems[0].name, 'The SHIELD Target',
             'the-system-tenants single system is SHIELD');

    systems = db.systems({ tenant: 'the-acme-tenant' });
    is.ok(systems, 'the-acme-tenant has targets/systems');
    is.equal(systems.length, 2, 'the-acme-tenant should have two systems');
    is.any(systems, { name: 'The CCDB Target' },
           'the-acme-tenant should have a CCDB target');
    is.any(systems, { name: 'The UAADB Target' },
           'the-acme-tenant should have a UAADB target');

    systems = db.systems({ tenant: 'a-non-existent-tenant' });
    is.empty(systems, 'a-non-existent-tenant has no targets/systems');

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
    var store, stores, db = Dataset();

    stores = db.stores({ tenant: 'the-system-tenant' })
    is.ok(stores, 'the-system-tenant has stores');
    is.equal(stores.length, 1, 'the-system-tenant has a single store');
    is.equal(stores[0].name, 'The System S3 Store',
             'the-system-tenants single store is System S3');

    stores = db.stores({ tenant: 'the-acme-tenant' });
    is.ok(stores, 'the-acme-tenant has targets/stores');
    is.empty(stores, 'the-acme-tenant should have no stores');

    stores = db.stores({ tenant: 'the-system-tenant',
                         global: true });
    is.equal(stores.length, 2, 'ths-system-tenant can access two stores');
    is.any(stores, { name: 'The System S3 Store' },
           'the-system-tenant should have access to the System S3 store');
    is.any(stores, { name: 'The Global Store' },
           'the-system-tenant should have access to the Global store');

    stores = db.stores({ tenant: 'a-non-existent-tenant' });
    is.empty(stores, 'a-non-existent-tenant has no stores');

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
  });

  QUnit.test('Job Retrieval', function (is) {
    var job, jobs, db = Dataset();

    jobs = db.jobs({ system: 'the-shield-target' });
    is.equal(jobs.length, 2, 'the-shield-target has two jobs');
    is.any(jobs, { name: 'Daily', schedule: 'daily 3:35am' },
           'the-shield-target has a daily (3:35am) job');
    is.any(jobs, { name: 'Weekly', schedule: 'weekly on sundays at 6:15am' },
           'the-shield-target has a weekly (sun 6:15am) job');

    jobs = db.jobs({ system: 'the-shield-target',
                     tenant: 'the-system-tenant' });
    is.equal(jobs.length, 2, 'the-shield-target (on the-system-tenant) has two jobs');
    is.any(jobs, { name: 'Daily', schedule: 'daily 3:35am' },
           'the-shield-target (on the-system-tenant) has a daily (3:35am) job');
    is.any(jobs, { name: 'Weekly', schedule: 'weekly on sundays at 6:15am' },
           'the-shield-target (on the-system-tenant) has a weekly (sun 6:15am) job');

    jobs = db.jobs({ system: 'the-shield-target',
                     tenant: 'the-acme-tenant' });
    is.empty(jobs, 'the-acme-tenant does not have the-shield-target as a system, so it can have no jobs for it');
  });

  QUnit.test('Task Retrieval', function (is) {
    var task, tasks, db = Dataset();

    tasks = db.tasks({ tenant: 'the-system-tenant' });
    is.set(tasks, [
      { uuid: 'test-system-s3-store-task-1' },
      { uuid: 'shield-backup-task-1' },
      { uuid: 'shield-backup-task-2' },
      { uuid: 'shield-backup-task-3' },
      { uuid: 'shield-purge-task-1'  } ],
      'the-system-tenant has four total tasks');

    tasks = db.tasks({ tenant: 'the-system-tenant',
                       system: 'the-shield-target' });
    is.set(tasks, [
      { uuid: 'shield-backup-task-1' },
      { uuid: 'shield-backup-task-2' },
      { uuid: 'shield-backup-task-3' } ],
      'the-system-tenant has three total tasks for system the-shield-target');

    tasks = db.tasks({ tenant: 'the-system-tenant',
                       system: 'the-shield-target',
                       job:    'the-shield-weekly-job' });
    is.set(tasks, [
      { uuid: 'shield-backup-task-3' } ],
      'the-system-tenant has one task for system the-shield-target (weekly job)');

    tasks = db.tasks({ tenant: 'the-system-tenant',
                       store:  'the-system-s3-store' });
    is.set(tasks, [
      { uuid: 'shield-backup-task-1' },
      { uuid: 'shield-backup-task-2' },
      { uuid: 'shield-purge-task-1'  },
      { uuid: 'test-system-s3-store-task-1' } ],
      'the-system-tenant has three tasks for system the-shield-target in the-system-s3-store');

    tasks = db.tasks({ tenant: 'the-system-tenant',
                       archive: 'shield-backup-archive-1' });
    is.set(tasks, [
      { uuid: 'shield-backup-task-1' },
      { uuid: 'shield-purge-task-1'  } ],
      'the-system-tenant has two tasks for system the-shield-target, archive shield-backup-archive-1');
  });

  QUnit.test('Archive Retrieval', function (is) {
    var archive, archives, db = Dataset();

    is.ok(true, 'TBD');

    /* by tenant */
    archives = db.archives({tenant: 'the-system-tenant' });
    is.set(archives, [
      { uuid: 'shield-backup-archive-1' },
      { uuid: 'shield-backup-archive-2' },
      { uuid: 'shield-backup-archive-3' } ],
      'the-system-tenant has three archives total');

    /* by tenant + target */
    archives = db.archives({ tenant: 'the-acme-tenant',
                             system: 'the-ccdb-target' });
    is.set(archives, [
      { uuid: 'ccdb-backup-archive-1' } ],
      'the-acme-tenant has one archive for the-ccdb-target, total');

    /* by tenant + target + store */
    archives = db.archives({ tenant: 'the-system-tenant',
                             system: 'the-shield-target',
                             store:  'the-system-s3-store'});
    is.set(archives, [
      { uuid: 'shield-backup-archive-1' },
      { uuid: 'shield-backup-archive-2' } ],
      'the-system-tenant has two archives for the-shield-target in the-system-s3-store');
  });
}
