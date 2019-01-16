if (typeof window.S           === 'undefined') { window.S           = {}; }
if (typeof window.S.H         === 'undefined') { window.S.H         = {}; }
if (typeof window.S.H.I       === 'undefined') { window.S.H.I       = {}; }
if (typeof window.S.H.I.E     === 'undefined') { window.S.H.I.E     = {}; }
if (typeof window.S.H.I.E.L   === 'undefined') { window.S.H.I.E.L   = {}; }
if (typeof window.S.H.I.E.L.D === 'undefined') { window.S.H.I.E.L.D = {}; }
window.S.H.I.E.L.D.Database = (function () {
  function Database(continuation) {
    var self = window.SHIELD = this;

    this._ = { data: {} };

    console.log('connecting to SHIELD event stream at /v2/events...');
    this._.ws = new WebSocket(document.location.protocol.replace(/http/, 'ws')+'//'+document.location.host+'/v2/events');

    this._.ws.onclose = function (event) {
      console.log('websocket closing...');
      if (continuation) {
        continuation(self);
      }
    };

    this._.ws.onmessage = function (m) {
      var update = {};

      try {
        update = JSON.parse(m.data);
      } catch (e) {
        console.log("unable to parse event '%s' from stream: ", m.data, e);
        return;
      }

      switch (update.event) {
      case 'create-object':
        self.set(update.type, update.data);

      //case 'update-object':
      }
    }

    this._.ws.onopen = function () {
      console.log('connected to event stream.');
      console.log('getting our bearings (via /v2/bearings)...');
      api({
        type: 'GET',
        url:  '/v2/bearings',
        success: function (bearings) {
          self._.shield = bearings.shield;
          self._.vault  = bearings.vault;
          self._.user   = bearings.user;
          self._.global = {
            stores: bearings.stores
          };

          self._.tasks   = {};
          self._.tenant  = self._.user.default_tenant;
          self._.tenants = bearings.tenants;
          for (var uuid in self._.tenants) {
            self._.tenants[uuid].archives = self.keyBy(self._.tenants[uuid].archives, 'uuid');
            self._.tenants[uuid].jobs     = self.keyBy(self._.tenants[uuid].jobs,     'uuid');
            self._.tenants[uuid].targets  = self.keyBy(self._.tenants[uuid].targets,  'uuid');
            self._.tenants[uuid].stores   = self.keyBy(self._.tenants[uuid].stores,   'uuid');
            for (var k in self._.tenants[uuid].tenant) {
              self._.tenants[uuid][k] = self._.tenants[uuid].tenant[k];
            }
            delete self._.tenants[uuid].tenant;
          }

          /* process grants... */
          self._.system = {};
          self._.system.grants = {
            admin:    false,
            manager:  false,
            engineer: false
          }
          switch (self._.user.sysrole) {
          case "admin":    self._.system.grants.admin    = true;
          case "manager":  self._.system.grants.manager  = true;
          case "engineer": self._.system.grants.engineer = true;
          }

          /* set default tenant */
          if (!self._.tenant) {
            tenants = self.sortBy(self.values(self._.tenants), 'name');
            if (tenants.length > 0) { self._.tenant = tenants[0]; }
          }

          self.redraw();
          var fn = continuation;
          continuation = undefined;
          fn(self);
        }
      });
    };
  }

  /*

  ##     ## ######## #### ##       #### ######## #### ########  ######
  ##     ##    ##     ##  ##        ##     ##     ##  ##       ##    ##
  ##     ##    ##     ##  ##        ##     ##     ##  ##       ##
  ##     ##    ##     ##  ##        ##     ##     ##  ######    ######
  ##     ##    ##     ##  ##        ##     ##     ##  ##             ##
  ##     ##    ##     ##  ##        ##     ##     ##  ##       ##    ##
   #######     ##    #### ######## ####    ##    #### ########  ######

   */

  /*
     _.merge(base, update) -> Object

     Perform a key-wise merge of `base` and `update`, overriding keys
     in `base` that are present in `update` with the values from `update`.

     This is a dumb merge at the moment; it does not recurse, and it does
     not compare lists item-wise.
   */
  Database.prototype.merge = function (base, update) {
    if (typeof(update) !== 'undefined') {
      for (var k in update) {
        if (update.hasOwnProperty(k)) {
          base[k] = update[k];
        }
      }
    }
    return base;
  };

  Database.prototype.keys = function (o) {
    var l = [];
    if (o) { for (var k in o) { l.push(k); } }
    return l;
  }

  Database.prototype.values = function (o) {
    var l = [];
    if (o) { for (var k in o) { l.push(o[k]); } }
    return l;
  }

  Database.prototype.sortBy = function (l, k) {
    return l.sort(function (a, b) {
      return a[k] > b[k] ? 1 : a[k] == b[k] ? 0 : -1;
    });
  };

  Database.prototype.keyBy = function (l,k) {
    var o = {};
    if (l) {
      for (var i = 0; i < l.length; i++) {
        o[l[i][k]] = l[i];
      }
    }
    return o;
  };

  Database.prototype.first = function (thing, fn) {
    if (typeof(thing) === 'undefined') {
      return undefined;
    }

    if (!thing instanceof Array) {
      console.log('first() called with a non-array first argument: ', thing);
      throw 'first() can only be used on arrays';
    }

    for (var i = 0; i < thing.length; i++) {
      if (fn(thing[i])) { return thing[i]; }
    }

    return undefined;
  };

  Database.prototype.each = function (thing, fn) {
    if (thing instanceof Array) {
      for (var i = 0; i < thing.length; i++) {
        fn.apply(this, [i, thing[i]], thing);
      }
      return
    }

    for (var k in thing) {
      if (thing.hasOwnProperty(k)) {
        fn.apply(this, [k, thing[k]], thing);
      }
    }
  };

  Database.prototype.map = function (thing, fn) {
    if (thing instanceof Array) {
      var l = [];
      for (var i = 0; i < thing.length; i++) {
        var x = fn.apply(this, [i, thing[i], thing]);
        if (typeof(x) !== 'undefined') {
          l.push(x);
        }
      }
      return l;
    }

    var o = {};
    for (var k in thing) {
      if (thing.hasOwnProperty(k)) {
        var x = fn.apply(this, [k, thing[k], thing]);
        if (typeof(x) !== 'undefined') {
          o[k] = x;
        }
      }
    }
    return o;
  };

  Database.prototype.clone = function (thing) {
    return this.map(thing, function (k,v) { return v; });
  };

  Database.prototype.diff = function (a, b) {
    if (typeof(a) === 'undefined' || typeof(b) === 'undefined') { return true; }
    return JSON.stringify(a) != JSON.stringify(b);
  }

  Database.prototype.is = function (role, context) {
    if (arguments.length == 1) {
      if (typeof(role) === 'object' && ('uuid' in role)) {
        return this._.user && this._.user.uuid == role.uuid;
      }
      return !!this._.system.grants[role];
    }
    if (typeof(context) === 'object' && ('uuid' in context)) {
      context = context.uuid;
    }
    if (this._.tenants[context]) {
      return !!this._.tenants[context].grants[role];
    }
    return false;
  };

  /*

  ########     ###    ########    ###        #######  ########   ######
  ##     ##   ## ##      ##      ## ##      ##     ## ##     ## ##    ##
  ##     ##  ##   ##     ##     ##   ##     ##     ## ##     ## ##
  ##     ## ##     ##    ##    ##     ##    ##     ## ########   ######
  ##     ## #########    ##    #########    ##     ## ##              ##
  ##     ## ##     ##    ##    ##     ##    ##     ## ##        ##    ##
  ########  ##     ##    ##    ##     ##     #######  ##         ######

  */

  Database.prototype._set = function (collection, idx, object) {
    if (!(idx in collection)) {
      collection[idx] = {};
    }
    if (object) {
      this.merge(collection[idx], object);
    }
  };

  Database.prototype.set = function (/* ... */) {
    if (arguments.length != 2 && arguments.length != 3) {
      console.log('set() called with the wrong number of arguments (want 2 or 3, but got %d): ', arguments.length, arguments);
      throw 'set() called with the wrong number of arguments';
    }

    var type = arguments[0],
        id, object;

    if (arguments.length == 2) {
      object = arguments[1];
      if (!('uuid' in object)) {
        console.log('set() [2-argument form] called with an object that does not have a UUID: ', object);
        throw 'set() called with a bad object (no UUID); either use the 3-argument form, or set the `uuid` property';
      }
      id = object.uuid;

    } else {
      id = arguments[1];
      object = arguments[2];
    };

    console.log('set(): updating object [%s %s] to be ', type, id, object);
    if (type == 'tenant') {
      this._set(this._.tenants, id, object);
      return;
    }

    if (type == 'task') {
      this._set(this._.tasks, id, object);
      return;
    }

    if (!('tenant_uuid' in object)) {
      console.log('unable to set object [%s %s]: object has no tenant_uuid: ', type, id, object);
      throw 'set() called with a bad object: '+type+' objects MUST have a `tenant_uuid` property';
    }
    if (!(object.tenant_uuid in this._.tenants)) {
      console.log('unable to set object [%s %s] on tenant "%s": tenant not found', type, id, object.tenant_uuid);
      return; /* this is just a warning... */
    }

    switch (type) {
    case 'archive': this._set(this._.tenants[object.tenant_uuid].archives, id, object); break;
    case 'job':     this._set(this._.tenants[object.tenant_uuid].jobs,     id, object); break;
    case 'target':  this._set(this._.tenants[object.tenant_uuid].targets,  id, object); break;
    case 'store':   this._set(this._.tenants[object.tenant_uuid].stores,   id, object); break;
    default:
      console.log('unable to set object [%s %s]: unrecognized type for object: ', type, id, object);
      throw 'set() called with a bad object: '+type+' is an unrecognized type';
    }
  };

  Database.prototype.unset = function (type, id) { /* FIXME */
    if (arguments.length != 2 && arguments.length != 3) {
      console.log('unset() called with the wrong number of arguments (want 2 or 3, but got %d): ', arguments.length, arguments);
      throw 'unset() called with the wrong number of arguments';
    }

    var type = arguments[0],
        id, object;

    if (arguments.length == 2) {
      object = arguments[1];
      if (!('uuid' in object)) {
        console.log('unset() [2-argument form] called with an object that does not have a UUID: ', object);
        throw 'unset() called with a bad object (no UUID); either use the 3-argument form, or set the `uuid` property';
      }
      id = object.uuid;

    } else {
      id = arguments[1];
      object = arguments[2];
    };

    if (type == 'tenant') {
      console.log('unset(): deleting object [%s %s]', type, id);
      delete this._.tenants[id];
      return;
    }

    if (type == 'task') {
      console.log('unset(): deleting object [%s %s]', type, id);
      delete this._.tasks[id];
      return;
    }

    if (!('tenant_uuid' in object)) {
      console.log('unable to delete object [%s %s]: object has no tenant_uuid: ', type, id, object);
      throw 'unset() called with a bad object: '+type+' objects MUST have a `tenant_uuid` property';
    }
    if (!(object.tenant_uuid in this._.tenants)) {
      console.log('unable to delete object [%s %s] on tenant %s: tenant not found', type, id, object.tenant_uuid);
      return; /* this is just a warning... */
    }

    console.log('unset(): deleting object [%s %s] from tenant %s', type, id, object.tenant_uuid);
    switch (type) {
    case 'archive': delete this._.tenants[object.tenant_uuid].archives[id]; break;
    case 'job':     delete this._.tenants[object.tenant_uuid].jobs[id];     break;
    case 'target':  delete this._.tenants[object.tenant_uuid].targets[id];  break;
    case 'store':   delete this._.tenants[object.tenant_uuid].stores[id];   break;
    default:
      console.log('unable to delete object [%s %s]: unrecognized type for object: ', type, id, object);
      throw 'unset() called with a bad object: '+type+' is an unrecognized type';
    }
  };




  Database.prototype.authenticated = function () {
    return typeof(this._.user) !== 'undefined';
  };

  Database.prototype.activeTenant = function () {
    if (this._.tenant && this._.tenants) {
      return this._.tenants[this._.tenant];
    }
    return undefined;
  };





  Database.prototype.systems = function () {
    var tenant = this.activeTenant();
    if (!tenant) {
      return [];
    }

    var systems = [];
    this.each(tenant.targets, function (uuid, target) {
      var system = this.clone(target);

      system.jobs = [];
      system.healthy = true;

      this.each(tenant.jobs, function (uuid, job) {
        if (system.uuid == job.target.uuid) {
          job = this.clone(job);
          //job.store = tenant.stores[job.store_uuid];

          if (!job.healthy) {
            system.healthy = false;
          }

          system.jobs.push(job);
        }
      });

      systems.push(system);
    });

    return systems;
  };

  Database.prototype.system = function (id) {
    var systems = this.systems(),
        found;

    this.each(systems, function (_, system) {
      if (found || system.uuid != id) { return; }
      found = system;
    });

    return found;
  };

  Database.prototype.stores = function (options) {
    options = this.merge({includeGlobal: true}, options);

    var tenant = this.activeTenant();
    if (!tenant) {
      return [];
    }

    var stores = [];
    if (options.includeGlobal) {
      stores = this.clone(this._.global.stores);
    }
    this.each(tenant.stores, function (uuid, store) {
      stores.push(this.clone(store));
    });

    return stores;
  };





  Database.prototype.redraw = function () {
    if (this.authenticated()) {
      $('#viewport').html(template('layout'));
    }

    $('#hud').html(template('hud'));
    $('.top-bar').html(template('top-bar'));
    document.title = "SHIELD "+this._.shield.env;
  };

  Database.prototype.health = function () {
    var h = {
      core:    this._.vault,
      storage: "ok",
      jobs:    "ok"
    };

    if (this.activeTenant()) {
      var tenant = this.activeTenant();
      for (var uuid in tenant.stores) {
        if (!tenant.stores[uuid].healthy) {
          console.log('HEALTH: storage system %s is failing!', tenant.stores[uuid].uuid);
          h.storage = "failing";
        }
      }

      for (var uuid in tenant.jobs) {
        if (!tenant.jobs[uuid].healthy) {
          console.log('HEALTH: job %s is failing!', tenant.jobs[uuid].uuid);
          h.jobs = "failing";
        }
      }
    };

    return h;
  }

  Database.prototype.stats = function () {
    var s = {
      jobs:     0,
      archives: 0,
      storage:  0,
      delta:    0
    };

    /* count our jobs */
    for (var uuid in this._.data.job) {
      if (this._.data.job[uuid].tenant_uuid == this._.tenant) {
        s.jobs++;
      }
    }

    /* count archives and storage footprint */
    for (var uuid in this._.data.archive) {
      if (this._.data.archive[uuid].tenant_uuid == this._.tenant) {
        s.archives++;
        s.storage += this._.data.archive[uuid].size;
      }
    }

    /* FIXME delta! */
    return s;
  };

  return Database;
})();
