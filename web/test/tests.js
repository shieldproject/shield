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
      /* TARGETS */
      .insert('target', {
        uuid: 'the-shield-target',
        name: 'The SHIELD Target'
      })
      .insert('target', {
        uuid: 'the-ccdb-target',
        name: 'The CCDB Target'
      })
      .insert('target', {
        uuid: 'the-uaadb-target',
        name: 'The UAADB Target'
      })

      /* STORES */
      .insert('store', {
        uuid: 'the-global-store',
        name: 'The Global Store',
        global: true
      })
      .insert('store', {
        uuid: 'local-bucket',
        name: 'The System S3 Store'
      })

      /* JOBS */
      .insert('job', {
        uuid: 'the-shield-daily-job',
        name: 'Daily',
        schedule: 'daily 3:35am',
        bucket:  'local-bucket',
        target_uuid: 'the-shield-target'
      })
      .insert('job', {
        uuid: 'the-shield-weekly-job',
        name: 'Weekly',
        schedule: 'weekly on sundays at 6:15am',
        bucket:  'the-global-store',
        target_uuid: 'the-shield-target'
      })
      .insert('job', {
        uuid: 'the-ccdb-hourly-job',
        name: 'Hourly',
        schedule: 'every hour at *:25',
        bucket:  'the-global-store',
        target_uuid: 'the-ccdb-target'
      })
      .insert('job', {
        uuid: 'the-uaadb-hourly-job',
        name: 'Hourly',
        schedule: 'every hour at *:25',
        bucket:  'the-global-store',
        target_uuid: 'the-uaadb-target'
      })

      /* TASKS */
      .insert('task', {
        uuid:   'shield-backup-task-1',
        op:     'backup',
        status: 'done',
        job_uuid:     'the-shield-daily-job',
        archive_uuid: 'shield-backup-archive-1',
        bucket:   'local-bucket',
        target_uuid:  'the-shield-target'
      })
      .insert('task', {
        uuid:   'shield-backup-task-2',
        op:     'backup',
        status: 'done',
        job_uuid:     'the-shield-daily-job',
        archive_uuid: 'shield-backup-archive-2',
        bucket:   'local-bucket',
        target_uuid:  'the-shield-target'
      })
      .insert('task', {
        uuid:   'shield-purge-task-1',
        op:     'purge',
        status: 'done',
        archive_uuid: 'shield-backup-archive-1',
        bucket:   'local-bucket'
      })
      .insert('task', {
        uuid:   'shield-backup-task-3',
        op:     'backup',
        status: 'done',
        job_uuid:     'the-shield-weekly-job',
        archive_uuid: 'shield-backup-archive-3',
        bucket:   'the-global-store',
        target_uuid:  'the-shield-target'
      })

      .insert('task', {
        uuid:   'ccdb-backup-task-1',
        op:     'backup',
        status: 'done',
        job_uuid:     'the-ccdb-hourly-job',
        archive_uuid: 'ccdb-backup-archive-1',
        bucket:   'the-global-store',
        target_uuid:  'the-ccdb-target'
      })

      .insert('task', {
        uuid:   'uaadb-backup-task-1',
        op:     'backup',
        status: 'done',
        job_uuid:     'the-uaadb-hourly-job',
        archive_uuid: 'uaadb-backup-archive-1',
        bucket:   'the-global-store',
        target_uuid:  'the-uaadb-target'
      })

      /* ARCHIVES */
      .insert('archive', {
        uuid: 'shield-backup-archive-1',
        target_uuid: 'the-shield-target',
        bucket:  'local-bucket',
        status:      'valid'
      })
      .insert('archive', {
        uuid: 'shield-backup-archive-2',
        target_uuid: 'the-shield-target',
        bucket:  'local-bucket',
        status:      'invalid'
      })
      .insert('archive', {
        uuid: 'shield-backup-archive-3',
        target_uuid: 'the-shield-target',
        bucket:  'the-global-store',
        status:      'purged'
      })
      .insert('archive', {
        uuid: 'ccdb-backup-archive-1',
        target_uuid: 'the-ccdb-target',
        bucket:  'the-global-store',
        status:      'purged'
      })
      .insert('archive', {
        uuid: 'uaadb-backup-archive-1',
        target_uuid: 'the-uaadb-target',
        bucket:  'the-global-store',
        status:      'valid'
      })
    ;
  };

  QUnit.test('System Retrieval', function (is) {
    var db = Dataset();

    is.set(db.systems(),
      [ { name: 'The SHIELD Target' },
        { name: 'The CCDB Target' },
        { name: 'The UAADB Target' } ],
      'there are three systems');

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

  QUnit.test('Job Retrieval', function (is) {
    var db = Dataset();

    is.set(db.jobs({ system: 'the-shield-target' }),
      [ { name: 'Daily',  schedule: 'daily 3:35am' },
        { name: 'Weekly', schedule: 'weekly on sundays at 6:15am' } ],
      'the-shield-target has a daily job (3:35am) and a weekly job (sun 6:15am)');

    is.set(db.jobs({ system: 'the-shield-target' }),
      [ { name: 'Daily',  schedule: 'daily 3:35am' },
        { name: 'Weekly', schedule: 'weekly on sundays at 6:15am' } ],
      'the-shield-target has a daily job (3:35am) and a weekly job (sun 6:15am)');

    /* single job retrieval */
    is.contained(
      db.job('the-shield-daily-job'),
      { target_uuid: 'the-shield-target',
        bucket:  'local-bucket' },
      'the-shield-daily-job can be retrieved');
    is.ok(!db.job('a-nonexistent-job'),
          'a non-existent-job cannot be retrieved');
  });

  QUnit.test('Task Retrieval', function (is) {
    var db = Dataset();

    is.set(db.tasks(),
      [ { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-backup-task-2' },
        { uuid: 'shield-backup-task-3' },
        { uuid: 'shield-purge-task-1'  },
        { uuid: 'ccdb-backup-task-1'   },
        { uuid: 'uaadb-backup-task-1'  },
      ], 'there are six total tasks');

    is.set(db.tasks({ system: 'the-shield-target' }),
      [ { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-backup-task-2' },
        { uuid: 'shield-backup-task-3' } ],
      'there are three total tasks for system the-shield-target');

    is.set(db.tasks({ system: 'the-shield-target',
                      job:    'the-shield-weekly-job' }),
      [ { uuid: 'shield-backup-task-3' } ],
      'there are one task for system the-shield-target (weekly job)');

    is.set(db.tasks({ archive: 'shield-backup-archive-1' }),
      [ { uuid: 'shield-backup-task-1' },
        { uuid: 'shield-purge-task-1'  } ],
      'there are two tasks for system the-shield-target, archive shield-backup-archive-1');

    /* single task retrieval */
    is.contained(
      db.task('shield-backup-task-1'),
      { target_uuid: 'the-shield-target',
        bucket:  'local-bucket' },
      'the shield-backup-task-1 task can be retrieved');
    is.ok(!db.task('a-nonexistent-task'),
          'a non-existent-task task cannot be retrieved');
  });

  QUnit.test('Archive Retrieval', function (is) {
    var db = Dataset();

    is.set(db.archives(),
      [ { uuid: 'shield-backup-archive-1' },
        { uuid: 'shield-backup-archive-2' },
        { uuid: 'shield-backup-archive-3' },
        { uuid: 'ccdb-backup-archive-1'   },
        { uuid: 'uaadb-backup-archive-1'  }
      ], 'there are five archives total');

    /* by target */
    is.set(db.archives({ system: 'the-ccdb-target' }),
      [ { uuid: 'ccdb-backup-archive-1' } ],
      'there is one archive for the-ccdb-target, total');

    /* by purged-less-ness */
    is.set(db.archives({ purged: false }),
      [ { uuid:   'shield-backup-archive-1' },
        { uuid:   'shield-backup-archive-2' },
        { uuid:   'uaadb-backup-archive-1' }
      ],
      'asking only for non-purged archives correctly retrieves valid and invalid archives, but not purged');

    /* by purged-ness */
    is.set(db.archives({ purged: true }),
      [ { uuid: "ccdb-backup-archive-1"   },
        { uuid: "shield-backup-archive-3" }
      ],
      'asking only for purged archives correctly retrieves only purged archives');

    /* single archive retrieval */
    is.contained(
      db.archive('shield-backup-archive-1'),
      { target_uuid: 'the-shield-target',
        bucket:  'local-bucket' },
      'the shield-backup-archive-1 archive can be retrieved');
    is.ok(!db.archive('a-nonexistent-archive'),
          'a non-existent-archive archive cannot be retrieved');
  });
})();

QUnit.module('AEGIS RBAC');
(function () {
  var Dataset = function () {
    return $.aegis();
  };

  QUnit.test('System-wide Roles', function (is) {
    var AEGIS = Dataset()

    AEGIS.grant('admin');
    is.equal(AEGIS.role(), 'Administrator');
    is.ok(AEGIS.is('admin'),    'A SHIELD administrator is considered an admin');
    is.ok(AEGIS.is('manager'),  'A SHIELD administrator is considered a manager');
    is.ok(AEGIS.is('engineer'), 'A SHIELD administrator is considered an engineer');
    is.ok(AEGIS.is('operator'), 'A SHIELD administrator is considered an operator');

    AEGIS.grant('manager');
    is.equal(AEGIS.role(), 'Manager');
    is.ok(!AEGIS.is('admin'),    'A SHIELD manager is NOT considered an admin');
    is.ok( AEGIS.is('manager'),  'A SHIELD manager is considered a manager');
    is.ok( AEGIS.is('engineer'), 'A SHIELD manager is considered an engineer');
    is.ok( AEGIS.is('operator'), 'A SHIELD administrator is considered an operator');

    AEGIS.grant('engineer');
    is.equal(AEGIS.role(), 'Engineer');
    is.ok(!AEGIS.is('admin'),    'A SHIELD engineer is NOT considered an admin');
    is.ok(!AEGIS.is('manager'),  'A SHIELD engineer is NOT considered a manager');
    is.ok( AEGIS.is('engineer'), 'A SHIELD engineer is considered an engineer');
    is.ok( AEGIS.is('operator'), 'A SHIELD engineer is considered an operator');

    AEGIS.grant('operator');
    is.equal(AEGIS.role(), 'Operator');
    is.ok(!AEGIS.is('admin'),    'A SHIELD operator is NOT considered an admin');
    is.ok(!AEGIS.is('manager'),  'A SHIELD operator is NOT considered a manager');
    is.ok(!AEGIS.is('engineer'), 'A SHIELD operator is NOT considered an engineer');
    is.ok( AEGIS.is('operator'), 'A SHIELD operator is considered an operator');

    AEGIS.grant('none');
    is.equal(AEGIS.role(), '');
    is.ok(!AEGIS.is('admin'),    'A SHIELD (nothing) is NOT considered an admin');
    is.ok(!AEGIS.is('manager'),  'A SHIELD (nothing) is NOT considered a manager');
    is.ok(!AEGIS.is('engineer'), 'A SHIELD (nothing) is NOT considered an engineer');
    is.ok(!AEGIS.is('operator'), 'A SHIELD (nothing) is NOT considered an operator');
  });
})();
