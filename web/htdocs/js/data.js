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
      system: {
        admin:    false,
        manager:  false,
        engineer: false,
        operator: false
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

    clear: function (type, object) {
      this.data = {};
      this.grants.system = {
        admin:    false,
        manager:  false,
        engineer: false,
        operator: false
      }
    },

    delete: function (type, object) {
      delete this.data[type][object.uuid];
    },

    find: function (type, query) {
      if (!(type in this.data)) { return undefined; }
      if ('uuid' in query) { return this.data[type][query.uuid]; }
      throw 'not implemented'; /* FIXME */
    },

    systems: function () {
      var systems = [];
      for (var uuid in this.data.target || {}) {
        var target = this.data.target[uuid];
        systems.push(target);
      }
      return systems;
    },
    system: function (uuid) {
      var target = this.find('target', { uuid: uuid });
      return target;
    },

    buckets: function () {
      var buckets = [];
      for (var key in this.data.bucket || {}) {
        buckets.push(this.data.bucket[key]);
      }
      return buckets;
    },
    bucket: function (key) {
      return this.find('bucket', { uuid: key });
    },

    jobs: function (q) {
      q = q || {};

      var jobs = [];
      for (var uuid in this.data.job || {}) {
        var job = this.data.job[uuid];
        if ('system' in q && job.target_uuid != q.system) {
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
        if (('system'  in q && task.target_uuid  != q.system)
         || ('job'     in q && task.job_uuid     != q.job)
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
      for (var u in this.data.agent || {}) {
        var agent = this.data.agent[u];
        if (!agent.hidden && (agent.uuid == uuid || agent.address == uuid)) {
          return agent;
        }
      }
      return undefined;
    },

    archives: function (q) {
      q = q || {};

      var archives = [];
        for (var uuid in this.data.archive || {}) {
          var archive = this.data.archive[uuid];
          if (('system'  in q && archive.target_uuid          != q.system)
           || ('purged'  in q && (archive.status == "purged") != q.purged)) {
            continue;
          }
          archives.push(archive);
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
      return uuid;
    },

    authenticated: function () {
      return typeof(this.user) !== 'undefined';
    },

    revoke: function () { // revoke all system rights
      this.grants.system.admin    = false;
      this.grants.system.manager  = false;
      this.grants.system.engineer = false;
      this.grants.system.operator = false;
      return this;
    },
    grant: function () { // grant a system rights
      this.revoke()
      switch (arguments[0]) {
      case 'admin':    this.grants.system.admin    = true;
      case 'manager':  this.grants.system.manager  = true;
      case 'engineer': this.grants.system.engineer = true;
      case 'operator': this.grants.system.operator = true;
      }
      return this;
    },
    role: function () {
      return this.grants.system.admin    ? 'Administrator'
           : this.grants.system.manager  ? 'Manager'
           : this.grants.system.engineer ? 'Engineer'
           : this.grants.system.operator ? 'Operator'
           : '';
    },
    is: function () {
      /* look up system rights: is($role) */
      return !!this.grants.system[arguments[0]];
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
        self.subscribe()
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
        case 'create-object': self.insert(update.type, update.data); break;
        case 'update-object': self.update(update.type, update.data); break;
        case 'delete-object': self.delete(update.type, update.data); break;
        case 'health-update': self.update(update.type, update.data); break;
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
        self.clear()
        console.log('getting our bearings (via %s)...', opts.bearings);
        api({
          type: 'GET',
          url:  opts.bearings,
          success: function (bearings) {
            self.shield = bearings.shield;
            self.user   = bearings.user;

            self.data.bucket = {};
            (bearings.buckets || []).forEach(bucket => {
              self.data.bucket[bucket.key] = bucket;
            });

            (bearings.archives || []).forEach(archive => {
              self.insert('archive', archive);
            });
            (bearings.jobs || []).forEach(job => {
              job.target_uuid = job.target.uuid;
              self.insert('job', job);
            });
            (bearings.targets || []).forEach(target => {
              self.insert('target', target);
            });
            (bearings.agents || []).forEach(agent => {
              self.insert('agent', agent);
            });

            /* process system grants... */
            self.grant(self.user.sysrole);
            df.resolve();
          },
          error: function () {
            df.reject();
          }
        });
      };

      return df.promise();
    },

    plugins: function () {
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

          list.push({
            id:     name,
            label:  plugin.name + ' (' + name + ')'
          });
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
