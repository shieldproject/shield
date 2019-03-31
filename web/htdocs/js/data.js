;(function ($, window, document, undefined) {

  /*

  ##     ## ######## #### ##       #### ######## #### ########  ######
  ##     ##    ##     ##  ##        ##     ##     ##  ##       ##    ##
  ##     ##    ##     ##  ##        ##     ##     ##  ##       ##
  ##     ##    ##     ##  ##        ##     ##     ##  ######    ######
  ##     ##    ##     ##  ##        ##     ##     ##  ##             ##
  ##     ##    ##     ##  ##        ##     ##     ##  ##       ##    ##
   #######     ##    #### ######## ####    ##    #### ########  ######

   */
  $.sortBy = function (l, k) {
    return l.sort(function (a, b) {
      return a[k] > b[k] ? 1 : a[k] == b[k] ? 0 : -1;
    });
  };

  $.first = function (thing, fn) {
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

  /*

   ######   ######  ########     ###    ########  ######  ##     ##
  ##    ## ##    ## ##     ##   ## ##      ##    ##    ## ##     ##
  ##       ##       ##     ##  ##   ##     ##    ##       ##     ##
   ######  ##       ########  ##     ##    ##    ##       #########
        ## ##       ##   ##   #########    ##    ##       ##     ##
  ##    ## ##    ## ##    ##  ##     ##    ##    ##    ## ##     ##
   ######   ######  ##     ## ##     ##    ##     ######  ##     ##

   */
  var Scratch = (function () {
    var scratch = {};

    var fn = function (k) {
      return scratch[k];
    };
    fn.clear = function () {
      scratch = {};
    };
    fn.track = function (k, v) {
      scratch[k] = v;
    };

    return fn;
  })();
  window.Scratch = Scratch;

  /*

     ###    ########  ######   ####  ######
    ## ##   ##       ##    ##   ##  ##    ##
   ##   ##  ##       ##         ##  ##
  ##     ## ######   ##   ####  ##   ######
  ######### ##       ##    ##   ##        ##
  ##     ## ##       ##    ##   ##  ##    ##
  ##     ## ########  ######   ####  ######

   */
  var AEGIS = function () {
    this.data = {};
    this.grants = {
      tenant: {},
      system: {
        admin:    false,
        manager:  false,
        engineer: false
      }
    };
  };

  $.extend(AEGIS.prototype, {
    insert: function (type, object) {
      if (!('uuid' in object)) { return undefined; }
      if (!(type in this.data)) { this.data[type] = {}; }
      this.data[type][object.uuid] = object;
      return this;
    },

    update: function (type, object) {
      if (!('uuid' in object)) { return undefined; }
      if (!(type in this.data)) { this.data[type] = {}; }
      if (!(object.uuid in this.data[type])) {
        this.data[type][object.uuid] = object;
      } else {
        for (var k in object) {
          this.data[type][object.uuid][k] = object[k];
        }
      }
      return this;
    },

    delete: function (type, object) {
      delete this.data[type][object.uuid];
    },

    find: function (type, query) {
      if (!(type in this.data)) { return undefined; }
      if ('uuid' in query) { return this.data[type][query.uuid]; }
      throw 'not implemented'; /* FIXME */
    },

    tenants: function () {

      var tenants = [];
      for (var uuid in this.data.tenant || {}) {
        var tenant = this.data.tenant[uuid];
        tenants.push(tenant);
      }
      return tenants;
    },
    tenant: function (uuid) {
      return this.find('tenant', { uuid: uuid });
    },

    systems: function (q) {
      q = q || {};

      var systems = [];
      if ('tenant' in q) {
        for (var uuid in this.data.target || {}) {
          if (this.data.target[uuid].tenant_uuid == q.tenant) {
            var target = this.data.target[uuid];
            target.healthy = $.all(this.jobs({ system: target.uuid }),
                                   function (j) { return j.healthy; });
            systems.push(target);
          }
        }
      }
      return systems;
    },
    system: function (uuid) {
      var target = this.find('target', { uuid: uuid });
      if (target) {
        target.healthy = $.all(this.jobs({ system: target.uuid }),
                               function (j) { return j.healthy; });
      }
      return target;
    },

    stores: function (q) {
      /* don't auto-vivify q */

      var stores = [];
      for (var uuid in this.data.store || {}) {
        var store = this.data.store[uuid];
        if (!q) {
          stores.push(store);

        } else if ('tenant' in q && store.tenant_uuid == q.tenant) {
          stores.push(store);

        } else if ('global' in q && store.global && q.global) {
          stores.push(store);
        }
      }
      return stores;
    },
    store: function (uuid) {
      return this.find('store', { uuid: uuid });
    },

    jobs: function (q) {
      q = q || {};

      var jobs = [];
      for (var uuid in this.data.job || {}) {
        var job = this.data.job[uuid];
        if ('system' in q && job.target_uuid != q.system) {
          continue;
        }
        if ('tenant' in q && job.tenant_uuid != q.tenant) {
          continue;
        }
        jobs.push(job);
      }
      return jobs;
    },
    job: function (uuid) {
      return this.find('job', { uuid: uuid });
    },

    tasks: function (q) {
      q = q || {};

      var tasks = [];
      for (var uuid in this.data.task || {}) {
        var task = this.data.task[uuid];
        if (('tenant'  in q && task.tenant_uuid != q.tenant)
         || ('system'  in q && task.target_uuid  != q.system)
         || ('job'     in q && task.job_uuid     != q.job)
         || ('store'   in q && task.store_uuid   != q.store)
         || ('archive' in q && task.archive_uuid != q.archive)) {
          continue;
        }
        tasks.push(task);
      }

      return tasks;
    },
    task: function (uuid) {
      return this.find('task', { uuid: uuid });
    },

    agents: function (q) {
      q = q || {};

      var agents = [];
      for (var uuid in this.data.agent || {}) {
        var agent = this.data.agent[uuid];
        if (!q.hidden && agent.hidden) {
          continue;
        }
        agents.push(agent);
      }

      return agents;
    },
    agent: function (uuid) {
      for (var uuid in this.data.agent || {}) {
        var agent = this.data.agent[uuid];
        if (!agent.hidden && (agent.uuid == uuid || agent.address == uuid)) {
          return agent;
        }
      }
      return undefined;
    },

    archives: function (q) {
      q = q || {};

      var archives = [];
      if ('tenant' in q) {
        for (var uuid in this.data.archive || {}) {
          var archive = this.data.archive[uuid];
          if (archive.tenant_uuid != q.tenant
           || ('system'  in q && archive.target_uuid  != q.system)
           || ('store'   in q && archive.store_uuid   != q.store)) {
            continue;
          }
          archives.push(archive);
        }
      }

      return archives;
    },
    archive: function (uuid) {
      return this.find('archive', { uuid: uuid });
    },

    uuid: function (uuid) {
      if (typeof(uuid) === 'object' && ('uuid' in uuid)) {
        uuid = uuid.uuid;
      }
      if (uuid === 'tenant') {
        return this.current ? this.current.uuid : '';
      }
      return uuid;
    },

    use: function (uuid) {
      this.current = this.tenant(uuid);
      return this;
    },

    locked: function () {
      return !!(this.vault && this.vault == "locked");
    },

    authenticated: function () {
      return typeof(this.user) !== 'undefined';
    },
    grant: function () {
      if (arguments.length == 1) {
        /* grant a system role: grant($role) */

        /* first revoke all explicit / implicit grants */
        this.grants.system.admin    = false;
        this.grants.system.manager  = false;
        this.grants.system.engineer = false;

        /* then grant only the privileges for this role */
        switch (arguments[0]) {
        case 'admin':    this.grants.system.admin    = true;
        case 'manager':  this.grants.system.manager  = true;
        case 'engineer': this.grants.system.engineer = true;
        }

      } else {
        /* grant a tenant role: grant($tenant, $role) */
        var uuid = arguments[0];

        /* first revoke all expliit / implicit grants */
        this.grants.tenant[uuid] = {
          admin:    false,
          engineer: false,
          operator: false
        };

        /* then grant only the privileges for this role */
        switch (arguments[1]) {
        case 'admin':    this.grants.tenant[uuid].admin    = true;
        case 'engineer': this.grants.tenant[uuid].engineer = true;
        case 'operator': this.grants.tenant[uuid].operator = true;
        }
      }
      return this;
    },
    role: function () {
      if (arguments.length == 0) {
        return this.grants.system.admin    ? 'Administrator'
             : this.grants.system.manager  ? 'Manager'
             : this.grants.system.engineer ? 'Engineer'
             : '';
      }

      var uuid = arguments[0];
      if (!(uuid in this.grants.tenant)) { return ''; }

      return this.grants.tenant[uuid].admin    ? 'Administrator'
           : this.grants.tenant[uuid].engineer ? 'Engineer'
           : this.grants.tenant[uuid].operator ? 'Operator'
           : '';
    },
    is: function () {
      if (arguments.length == 1) {
        /* look up system rights: is($role) */
        return !!this.grants.system[arguments[0]];

      } else {
        /* look up tenant rights: is($tenant, $role) */
        var uuid = arguments[0];
        if (uuid === 'tenant') {
          uuid = this.current;
        }
        if (typeof(uuid) === 'object' && 'uuid' in uuid) {
          uuid = uuid.uuid;
        }
        if (uuid in this.grants.tenant) {
          return !!this.grants.tenant[uuid][arguments[1]];
        }
        return false;
      }
    },

    subscribe: function (opts) {
      opts = $.extend({
        bearings:  '/v2/bearings',
        websocket: document.location.protocol.replace(/http/, 'ws')+'//'+document.location.host+'/v2/events'
      }, opts || {});

      var df = $.Deferred();
      var self = this; /* save off 'this' for the continuation call */

      console.log('connecting to websocket at %s', opts.websocket);
      this.ws = new WebSocket(opts.websocket);
      this.ws.onerror = function (event) {
        self.ws = undefined;
        console.log('websocket failed: ', event);
        df.reject();
      };
      this.ws.onclose = function () {
        self.ws = undefined;
        df.reject();
      };

      this.ws.onmessage = function (m) {
        var update = {};

        try {
          update = JSON.parse(m.data);
        } catch (e) {
          console.log("unable to parse event '%s' from stream: ", m.data, e);
          return;
        }

        //console.log('event (%s): ', update.event, JSON.stringify(update.data));
        switch (update.event) {
        case 'lock-core':     self.vault = "locked";   $('#hud').template('hud');
                                                       $('#lock-state').fadeIn(); break
        case 'unlock-core':   self.vault = "unlocked"; $('#hud').template('hud');
                                                       $('#lock-state').fadeOut(); break
        case 'create-object': self.insert(update.type, update.data); break;
        case 'update-object': self.update(update.type, update.data); break;
        case 'delete-object': self.delete(update.type, update.data); break;
        case 'task-log-update':
          var task = self.task(update.data.uuid);
          if (task) {
            if (!task.log) { task.log = ''; }
            task.log += update.data.tail; }
          break;
        case 'task-status-update':
          self.update('task', update.data);
          break;
        default:
          console.log('unrecognized websocket message "%s": %s', update.event, JSON.stringify(update.data));
          return;
        }

        if (Scratch('redrawable') && $('#main').is(':visible')) {
          $('#main').template();
        }
      };

      this.ws.onopen = function () {
        console.log('connected to event stream.');
        console.log('getting our bearings (via %s)...', opts.bearings);
        api({
          type: 'GET',
          url:  opts.bearings,
          success: function (bearings) {
            self.shield = bearings.shield;
            self.vault  = bearings.vault;
            self.user   = bearings.user;

            for (var i = 0; i < bearings.stores.length; i++) {
              self.insert('store', bearings.stores[i]);
            }
            for (var uuid in bearings.tenants) {
              var tenant = bearings.tenants[uuid];

              self.grant(uuid, tenant.role); /* FIXME: we don't need .grants anymore... */

              for (var i = 0; i < tenant.archives.length; i++) {
                self.insert('archive', tenant.archives[i]);
              }

              for (var i = 0; i < tenant.jobs.length; i++) {
                tenant.jobs[i].tenant_uuid = uuid;
                tenant.jobs[i].store_uuid = tenant.jobs[i].store.uuid;
                delete tenant.jobs[i].store;

                tenant.jobs[i].target_uuid = tenant.jobs[i].target.uuid;
                delete tenant.jobs[i].target;

                self.insert('job', tenant.jobs[i]);
              }

              for (var i = 0; i < tenant.targets.length; i++) {
                tenant.targets[i].tenant_uuid = uuid;
                self.insert('target', tenant.targets[i]);
              }

              for (var i = 0; i < tenant.stores.length; i++) {
                tenant.stores[i].tenant_uuid = uuid;
                self.insert('store', tenant.stores[i]);
              }

              for (var i = 0; i < tenant.agents.length; i++) {
                self.insert('agent', tenant.agents[i]);
              }
              delete tenant.agents;

              self.insert('tenant', tenant.tenant);
            }
            console.log(bearings);

            /* process system grants... */
            self.grant(self.user.sysrole);

            /* set default tenant */
            if (!self.current && self.data.tenant) {
              self.current = self.data.tenant[self.user.default_tenant];
            }
            if (!self.current) {
              var l = [];
              for (var k in self.data.tenant) {
                l.push(self.data.tenant[k]);
              }
              l.sort(function (a, b) {
                return a.name > b.name ? 1 : a.name == b.name ? 0 : -1;
              });
              if (l.length > 0) { self.current = l[0]; }
            }

            df.resolve();
          },
          error: function () {
            df.reject();
          }
        });
      };

      return df.promise();
    },

    plugins: function (type) {
      var seen = {}; /* de-duplicating map */
      var list = []; /* sorted list of [label, name] tuples */
      var map  = {}; /* plugin.id => [agent, ...] */

      $.each(this.agents({ hidden: false }), function (i, agent) {
        if (!agent.metadata || !agent.metadata.plugins) { return; }

        /* enumerate the plugins, and wire up the `map` associations */
        $.each(agent.metadata.plugins, function (name, plugin) {
          /* track that this plugin can be used on this agent */
          if (!(name in map)) { map[name] = []; }
          map[name].push(agent);

          /* if we've already seen this plugin, don't add it to the list
             that we will use to render the dropdowns; this does mean that,
             in the event of conflicting names for the same plugin (i.e.
             different versions of the same plugin), we may not have a
             deterministic sort, but ¯\_(ツ)_/¯
           */
          if (name in seen) { return; }
          seen[name] = true;

          if (plugin.features[type] == "yes") {
            list.push({
              id:     name,
              label:  plugin.name + ' (' + name + ')'
            });
          }
        });
      });

      /* sort plugin lists by metadata name ("Amazon" instead of "s3") */
      list.sort(function (a, b) {
        return a.label > b.label ?  1 :
               a.label < b.label ? -1 : 0;
      });

      return {
        list:   list,
        agents: map
      };
    }
  });

  $.aegis = function () {
    return new AEGIS();
  }
})(jQuery, window, document);
