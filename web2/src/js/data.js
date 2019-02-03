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
            self._.tenants[uuid].agents   = self.keyBy(self._.tenants[uuid].agents,   'uuid');
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
        fn.apply(this, [i, thing[i]]);
      }
      return
    }

    for (var k in thing) {
      if (thing.hasOwnProperty(k)) {
        fn.apply(this, [k, thing[k]]);
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
    if (!collection) {
      return;
    }
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


  Database.prototype.plugins = function (type) {
    var seen = {}; /* de-duplicating map */
    var list = []; /* sorted list of [label, name] tuples */
    var map  = {}; /* plugin.id => [agent, ...] */

    var tenant = this.activeTenant();
    if (!tenant) {
      return undefined;
    }

    for (var agent_uuid in tenant.agents) {
      var agent = tenant.agents[agent_uuid];
      agent.plugins = agent.metadata.plugins;

      /* enumerate the plugins, and wire up the `map` associations */
      for (var name in agent.plugins) {
        var plugin = agent.plugins[name];

        /* track that this plugin can be used on this agent */
        if (!(name in map)) { map[name] = []; }
        map[name].push(agent);

        /* if we've already seen this plugin, don't add it to the list
           that we will use to render the dropdowns; this does mean that,
           in the event of conflicting names for the same plugin (i.e.
           different versions of the same plugin), we may not have a
           deterministic sort, but ¯\_(ツ)_/¯
         */
        if (name in seen) { continue; }
        seen[name] = true;

        if (plugin.features[type] == "yes") {
          list.push({
            id:     name,
            label:  plugin.name + ' (' + name + ')'
          });
        }
      }
    }

    /* sort plugin lists by metadata name ("Amazon" instead of "s3") */
    list.sort(function (a, b) {
      return a.label > b.label ?  1 :
             a.label < b.label ? -1 : 0;
    });

    return {
      list:   list,
      agents: map
    };
  };


  Database.prototype.agent = function (id) {
    var tenant = this.activeTenant();
    if (!tenant) {
      return undefined;
    }

    if (!(id in tenant.agents)) {
      for (var uuid in tenant.agents) {
        if (tenant.agents[uuid].address == id) {
          return tenant.agents[uuid];
        }
      }
    }
    return tenant.agents[id];
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

      system.archives = [];
      this.each(tenant.archives, function (i, archive) {
        if (archive.target_uuid == system.uuid && archive.status == 'valid') {
          system.archives.push(archive);
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

  Database.prototype.findArchive = function (id, filter) {
    if (!filter) {
      filter = {};
    }

    var tenant = this.activeTenant();
    if (!tenant) {
      return undefined;
    }

    var found;
    this.each(tenant.archives, function (_, archive) {
      if (found || archive.uuid != id) { return; }
      found = archive;
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

  Database.prototype.store = function (id, options) {
    var stores = this.stores(options),
        found;

    this.each(stores, function (_, store) {
      if (found || store.uuid != id) { return; }
      found = store;
    });

    return found;
  };

  Database.prototype.storesForTarget = function (id) {
    var system = this.system(id),
        stores = [],
        seen   = {};

    this.each(this.stores(), function (_, store) {
      this.each(system.jobs, function (_, job) {
        if (job.store.uuid == store.uuid && !seen[store.uuid]) {
          stores.push(store);
          seen[store.uuid] = true;
        }
      });
    });

    return stores;
  };





  Database.prototype.redraw = function () {
    if (this.authenticated()) {
      $('#viewport').template('layout');
    }

    $('#hud').template('hud');
    $('.top-bar').template('top-bar');
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
