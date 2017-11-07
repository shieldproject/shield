var referer = undefined;
function divert(page) { // {{{
  if ($global.auth.unauthenticated) {
    if (page.match(/^#!\/(login|logout|cliauth)$/)) {
      return page;
    }
    console.log('session not authenticated; diverting to #!/login page...');
    return "#!/login";

  } else if ($global.auth.is.system.engineer && $global.hud) {
    /* process 'system' team diverts */
    if ($global.hud.health.core == "uninitialized") {
      console.log('system user detected, and this SHIELD core is uninitialized; diverting to #!/init page...');
      return "#!/init";

    } else if ($global.hud.health.core == "sealed" || $global.hud.health.core == "locked") {
      console.log('system user detected, and this SHIELD core is locked; diverting to #!/unlock page...');
      return "#!/unlock";
    }
  }
  if (!page || page == "") {
    return $global.auth.is.system.engineer ? '#!/admin' : '#!/systems';
  }

  return page;
}
// }}}

function dispatch(page) {
  var argv = page.split(/[:+]/);
  dest = argv.shift();
  page = divert(dest);
  args = {};
  for (var i = 0; i < argv.length; i += 2) {
    args[argv[0+i]] = argv[1+i]
  }

  console.log('dispatching to %s (from %s)...', page, dest);

  var top = page.replace(/^(#!\/[^\/]+)(\/.*)?$/, '$1');
  $('nav li.current').removeClass('current');
  $('nav a[href="'+top+'"]').closest('li').addClass('current');

  switch (page) {

  case "#!/login": /* {{{ */
    (function () {
      var progress = function (how) {
        $('#viewport').find('#logging-in').remove();
        $('#viewport').append(template('logging-in', {auth: how}));
      }

      api({
        type: 'GET',
        url:  '/v2/auth/providers?for=web',
        success: function (data) {
          $('#viewport').html($(template('login', { providers: data })))
            .on("click", ".login", function (event) {
              progress($(event.target).text());
            })
            .on("submit", "form", function (event) {
              event.preventDefault()
              progress('local SHIELD authentication');

              var $form = $(event.target);
              var data = $form.serializeObject();
              $form.reset();

              api({
                type: "POST",
                url:  "/v2/auth/login",
                data: data,
                success: function () {
                  /* this makes the chrome re-render unnecessary */
                  document.location.href = "/"
                },
                error: function (xhr) {
                  $(event.target).error(xhr.responseJSON);
                },
                complete: function () {
                  $('#viewport').find('#logging-in').remove();
                  //using the systems page as our landing page when a user logs in
                  document.location.href = "/#!/systems"
                }
              });
            });
         },
        error: function (xhr) {
          $('#viewport').html(template('BOOM'));
        }
      })
    })();
    break; /* #!/login */
    // }}}
  case "#!/cliauth": /* {{{ */
    $('#viewport').html(template('cliauth', args));
    break; /* #!/cliauth */
    // }}}
  case "#!/logout": /* {{{ */
    (function () {
      api({
        type: "GET",
        url: "/v2/auth/logout",
        success: function () {
          document.location.href = '/';
        },
        error: function (xhr) {
          if (xhr.status >= 500) {
            $('#viewport').html(template('BOOM'));
          } else {
            document.location.href = '/';
          }
        }
      });
    })()
    break;
    // }}}

  case "#!/do/backup": /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].operator) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'operator' }));
      break;
    }
    (function () {
      $('#main').html(template('loading'));
      var rerender = function (data) {
        $('#main').html($(template('do-backup', data))
          .on('click', "a[href^=\"do-backup:\"]", function (event) {
            event.preventDefault();
            l = $(event.target).closest('a[href^="do-backup:"]').attr('href').split(':');

            /* unpick      - unpick the target / store / policy {{{ */
            if (l[1] == "unpick") {
              var $card = $(event.target).closest('.card');
              if ($card.is('.job'))    { data.system = data.store = data.policy = undefined; }
              if ($card.is('.store'))  {               data.store = data.policy = undefined; }
              if ($card.is('.policy')) {                            data.policy = undefined; }

              data.noauto = true;
              rerender(data);
              data.noauto = false;
              return;
            }

            /* }}} */
            /* pick:target - pick the target we want to back up {{{ */
            if (l[1] == "pick" && l[2] == "target") {
              for (var i = 0; i < data.systems.length; i++) {
                if (data.systems[i].uuid == l[3]) {
                  data.system = data.systems[i];
                  break;
                }
              }
              rerender(data);
              return;
            }

            /* }}} */
            /* pick:store  - pick the store we want to store the backups in {{{ */
            if (l[1] == "pick" && l[2] == "store") {
              for (var i = 0; i < data.system.jobs.length; i++) {
                if (data.system.jobs[i].store.uuid == l[3]) {
                  data.store = data.system.jobs[i].store;
                  break;
                }
              }
              rerender(data);
              return;
            }

            /* }}} */
            /* pick:policy - pick the retention polict we want {{{ */
            if (l[1] == "pick" && l[2] == "policy") {
              for (var i = 0; i < data.system.jobs.length; i++) {
                if (data.system.jobs[i].store.uuid == data.store.uuid
                && data.system.jobs[i].retention.uuid == l[3]) {
                  data.policy = data.system.jobs[i].retention;
                  break;
                }
              }
              rerender(data);
              return;
            }
            /* }}} */
          })
          .on('click', 'button[rel^="run:"]', function (event) {
            console.log('local handler triggered');
            event.preventDefault(); event.stopPropagation();

            var uuid = $(event.target).attr('rel').replace(/^run:/, '');
            banner('Scheduling ad hoc backup...', 'progress');
            api({
              type: 'POST',
              url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/jobs/'+uuid+'/run',
              success: function () {
                banner('Ad hoc backup job scheduled');
                goto('#!/systems/system:uuid:'+data.system.uuid);
              },
              error: function () {
                banner('Unable to schedule ad hoc backup job', 'error');
              }
            });
          }));
      };

      api({
        type: 'GET',
        url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/systems',
        error: "Failed retrieving the list of protected systems from the SHIELD API.",
        success: function (systems) {
          rerender({ systems: systems });
        }
      });
    })();
    break; /* #!/do/backup */
    // }}}
  case "#!/do/restore": /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].operator) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'operator' }));
      break;
    }
    (function () {
      $('#main').html(template('loading'));
      var rerender = function (data) {
        $('#main').html($(template('do-restore', data))
          .on('click', 'a[href^="do-restore:"], button[rel^="do-restore:"]', function (event) {
            event.preventDefault();
            var l = [];
            if ($(event.target).is('a, a *')) {
              l = $(event.target).closest('a[href^="do-restore:"]').attr('href').split(':');
            } else {
              l = $(event.target).closest('button[rel^="do-restore:"]').attr('rel').split(':');
            }

            /* unpick       - unpick the target / store / policy {{{ */
            if (l[1] == "unpick") {
              var $card = $(event.target).closest('.card');
              if ($card.is('.job'))     { data.system = data.archive = undefined; }
              if ($card.is('.archive')) {               data.archive = undefined; }

              rerender(data);
              return;
            }

            /* }}} */
            /* pick:target  - pick the target we want to back up {{{ */
            if (l[1] == "pick" && l[2] == "target") {
              for (var i = 0; i < data.systems.length; i++) {
                if (data.systems[i].uuid == l[3]) {
                  data.system = data.systems[i];
                  break;
                }
              }
              rerender(data);
              return;
            }

            /* }}} */
            /* pick:archive - pick the store we want to store the restores in {{{ */
            if (l[1] == "pick" && l[2] == "archive") {
              for (var i = 0; i < data.archives.length; i++) {
                if (data.archives[i].uuid == l[3]) {
                  data.archive = data.archives[i];
                  break;
                }
              }
              rerender(data);
              return;
            }

            console.log('unhandled action: %s', l.join(":"));

            /* }}} */
            /* final        - run the restore {{{ */
            if (l[1] == "final") {
              banner('Scheduling restore...', 'progress');
              api({
                type: 'POST',
                url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/archives/'+data.archive.uuid+'/restore',
                success: function () {
                  banner('Ad hoc restore task scheduled');
                  goto('#!/systems/system:uuid:'+data.system.uuid);
                },
                error: function () {
                  banner('Unable to schedule ad hoc restore task', 'error');
                }
              });
            }

            /* }}} */

            console.log('unhandled action: %s', l.join(":"));
          }));
      };

      apis({
        base: '/v2/tenants/'+$global.auth.tenant.uuid,
        multiplex: {
          systems:  { type: 'GET', url: '+/systems' },
          archives: { type: 'GET', url: '+/archives' }
        },
        error: "Failed retrieving the list of protected systems from the SHIELD API.",
        success: rerender
      });
    })();
    break; /* #!/do/restore */
    // }}}
  case "#!/do/configure": /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    (function () {
      if (!$global.auth.tenant) {
        $('#main').html(template('you-have-no-tenants'));
        return;
      }
      $('#main').html(template('loading'));
      var data = {};

      var rerender = function (data) {
        $('#main').html($(template('do-configure', data))
          .on('click', 'a[href^="do-configure:"]', function (event) {
            event.preventDefault();
            l = $(event.target).closest('a[href^="do-configure:"]').attr('href').split(':');

            /* unpick: go back to a previous step {{{ */
            if (l[1] == "unpick") {
              var what = l[2];
              if (!what) {
                var $card = $(event.target).closest('.card');
                if ($card.is('.job'))    { what = "target";   }
                if ($card.is('.store'))  { what = "store";    }
                if ($card.is('.policy')) { what = "schedule"; } /* .card.policy is combo schedule+policy */
              }

              switch (what) {
              case "target"   : data.target = data.schedule = data.policy = data.store = undefined; break;
              case "schedule" :               data.schedule = data.policy = data.store = undefined; break;
              case "policy"   :                               data.policy = data.store = undefined; break;
              case "store"    :                                             data.store = undefined; break;
              }

              if (l[3] == 'redo') {
                switch (what) {
                case "target" : data.new_target = false; break;
                case "store"  : data.new_store  = false; break;
                case "policy" : data.new_policy = false; break;
                }
              }
              rerender(data);
              return;
            }

            /* }}} */
            /* pick:target - pick a target and render the next step: scheduling {{{ */
            if (l[1] == "pick" && l[2] == "target") {
              delete data.new_target;
              for (var i = 0; i < data.targets.length; i++) {
                if (data.targets[i].uuid == l[3]) {
                  data.target = data.targets[i];
                  break;
                }
              }

              rerender(data);
              return;
            }

            /* }}} */
            /* new:target  - show the 'new target' form {{{ */
            if (l[1] == "new" && l[2] == "target") {
              data.new_target = true;
              $('#main .step').html(template('loading'));
              api({
                type: 'GET',
                url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/agents',
                error: "Unable to retreive list of SHIELD Agents from the SHIELD API.",
                success: function (agents) {
                  data.agents = agents;
                  rerender(data);
                }
              });
              return;
            }

            /* }}} */
            /* pick:policy - pick a retention policy and render the next step: storage {{{ */
            if (l[1] == "pick" && l[2] == "policy") {
              delete data.new_policy;
              for (var i = 0; i < data.policies.length; i++) {
                if (data.policies[i].uuid == l[3]) {
                  data.policy = data.policies[i];
                  break;
                }
              }

              rerender(data);
              return;
            }

            /* }}} */
            /* new:policy  - show the 'new policy' form {{{ */
            if (l[1] == "new" && l[2] == "policy") {
              data.new_policy = true;
              rerender(data);
              return;
            }

            /* }}} */
            /* pick:store  - pick a store and render the next step: review {{{ */
            if (l[1] == "pick" && l[2] == "store") {
              delete data.new_store;
              for (var i = 0; i < data.stores.length; i++) {
                if (data.stores[i].uuid == l[3]) {
                  data.store = data.stores[i];
                  break;
                }
              }

              rerender(data);
              return;
            }

            /* }}} */
            /* new:store   - show the 'new store' form {{{ */
            if (l[1] == "new" && l[2] == "store") {
              data.new_store = true;
              $('#main .step').html(template('loading'));
              api({
                type: 'GET',
                url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/agents',
                error: "Unable to retreive list of SHIELD Agents from the SHIELD API.",
                success: function (agents) {
                  data.agents = agents;
                  rerender(data);
                }
              });
              return;
            }

            /* }}} */
          })
          .on('submit', 'form[action^="do-configure:"]', function (event) {
            event.preventDefault();
            var $form = $(event.target);
            var l = $form.attr('action').split(':');

            /* make:target   - validate a new target, store it for later {{{ */
            if (l[1] == 'make' && l[2] == 'target') {
              var target = $form.serializePluginForm();
              if (!$form.reset().validate(target).isOK()) { return; }
              api({
                type: 'POST',
                url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/targets?test=t',
                data: target,
                error: function (xhr) {
                  console.log('do-configure: target validation failed: ', xhr.responseJSON);
                  $form.error(xhr.responseJSON);
                },
                success: function (ok) {
                  data.target = target;
                  rerender(data);
                }
              });
              return;
            }

            /* }}} */
            /* make:schedule - validate and set the backup schedule {{{ */
            if (l[1] == 'make' && l[2] == 'schedule') {
              var spec = $form.timespec();
              $form.reset();
              api({
                type: 'POST',
                url:  '/v2/ui/check/timespec',
                data: { timespec: spec },
                error: function (xhr) {
                  $form.error(xhr.responseJSON);
                },
                success: function (rs) {
                  data.schedule = rs.ok
                  data.name = $('.scheduling .optgroup .selected[data-subform]').text();
                  rerender(data);
                }
              })
              return;
            }

            /* }}} */
            /* make:policy   - validate a new policy, store it for later {{{ */
            if (l[1] == 'make' && l[2] == 'policy') {
              var policy = $form.serializeObject();
              policy.expires = policy.days * 86400; /* FIXME */
              $form.reset();
              api({
                type: 'POST',
                url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies?test=t',
                data: policy,
                error: function (xhr) {
                  $form.error(xhr);
                },
                success: function (ok) {
                  data.policy = policy;
                  rerender(data);
                }
              });
              return;
            }

            /* }}} */
            /* make:store    - validate a new store, store it for later {{{ */
            if (l[1] == 'make' && l[2] == 'store') {
              var store = $form.serializePluginForm();
              if (!$form.reset().validate(store).isOK()) { return; }
              api({
                type: 'POST',
                url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores?test=t',
                data: store,
                error: function (xhr) {
                  $form.error(xhr);
                },
                success: function (ok) {
                  data.store = store;
                  rerender(data);
                }
              });
              return;
            }
            /* }}} */
            /* finalize      - create all the things! {{{ */
            if (l[1] == 'finalize') {
              var finalize = function () {
                console.log('finalizing....');
                if (!data.target.uuid) {
                  console.log('creating new target "%s"...', data.target.name);
                  console.dir(data.target);
                  api({
                    type: 'POST',
                    url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/targets',
                    data: data.target,
                    error: "Unable to create new data system",
                    success: function (ok) {
                      data.target.uuid = ok.uuid;
                      finalize();
                    }
                  });
                  return;
                }

                if (!data.policy.uuid) {
                  console.log('creating new policy "%s"...', data.policy.name);
                  console.dir(data.policy);
                  api({
                    type: 'POST',
                    url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies',
                    data: data.policy,
                    error: "Unable to create new retention policy",
                    success: function (ok) {
                      data.policy.uuid = ok.uuid;
                      finalize();
                    }
                  });
                  return;
                }

                if (!data.store.uuid) {
                  console.log('creating new store "%s"...', data.store.name);
                  console.dir(data.store);
                  api({
                    type: 'POST',
                    url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores',
                    data: data.store,
                    error: "Unable to create new cloud storage system",
                    success: function (ok) {
                      data.store.uuid = ok.uuid;
                      finalize();
                    }
                  });
                  return;
                }

                console.log('creating job...');
                api({
                  type: 'POST',
                  url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/jobs',
                  data: {
                    name     : 'a random name?', // FIXME
                    summary  : '',
                    schedule : data.schedule,

                    store    : data.store.uuid,
                    target   : data.target.uuid,
                    policy   : data.policy.uuid
                  },
                  error: "Unable to create a new backup job",
                  success: function (ok) {
                    goto("#!/systems");
                  }
                });
              };
              finalize();
              return;
            }
            /* }}} */
          }));
        $('#main .optgroup').optgroup(); $('.scheduling').subform();
        $('#main .scheduling [data-subform=schedule-daily]').trigger('click');

        $('#main [action="do-configure:make:target"]').pluginForm({ type: 'target' });
        $('#main [action="do-configure:make:store"]').pluginForm({ type: 'store' });
      };

      apis({
        base: '/v2/tenants/'+$global.auth.tenant.uuid,
        multiplex: {
          targets:  { type: 'GET', url: '+/targets'  },
          stores:   { type: 'GET', url: '+/stores'   },
          policies: { type: 'GET', url: '+/policies' }
        },
        success: rerender
      });
    })();
    break; /* #!/do/configure */
    // }}}

  case "#!/systems": /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/systems',
      error: "Failed retrieving the list of protected systems from the SHIELD API.",
      success: function (data) {
        $('#main').html(template('systems', { systems: data }));
      }
    });
    break; /* #!/systems */
    // }}}
  case '#!/systems/system': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/systems/'+args.uuid,
      error: "Failed retrieving metadata for protected system from the SHIELD API.",
      success: function (data) {
        $('#main').html(template('system', { target: data }));
      }
    });
    break; /* #!/systems/system */
    // }}}

  case '#!/stores': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    $('#main').html(template('loading'));
    apis({
      multiplex: {
        local:  { type: 'GET', url: '/v2/tenants/'+$global.auth.tenant.uuid+'/stores' },
        global: { type: 'GET', url: '/v2/global/stores' }
      },
      error: "Failed retrieving the list of storage endpoints from the SHIELD API.",
      success: function (data) {
        /* FIXME fixups that need to migrate into the SHIELD code */
        for (key in data.local)  { if (!('ok' in data.local[key]))  { data.local[key].ok  = true; } }
        for (key in data.global) { if (!('ok' in data.global[key])) { data.global[key].ok = true; } }
        $('#main').html(template('stores', { stores: $.extend({}, data.local, data.global) }));
      }
    });
    break; /* #!/stores */
    // }}}
  case '#!/stores/store': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    $('#main').html(template('loading'));

    var rerender = function (data) {
      /* FIXME fixups that need to migrate into the SHIELD code */
      data.ok = true;
      data.archives = data.archive_count;
      data.used = data.storage_used;
      data.projected = 2.1;
      data.daily_delta = data.daily_increase;
      $('#main').html(template('store', { store: data }));
    };
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores/'+args.uuid,
      success: rerender,
      error: function (xhr) {
        api({
          type: 'GET',
          url:  '/v2/global/stores/'+args.uuid,
          error: "Unable to retrieve storage systems from SHIELD API.",
          success: rerender
        });
      }
    });
    break; /* #!/stores/store */
    // }}}
  case '#!/stores/new': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/agents',
      error: "Unable to retrieve list of SHIELD Agents from the SHIELD API",
      success: function (data) {
        var cache = {};

        $('#main').html($(template('stores-form', { agents: data }))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializePluginForm();
            if (!$form.reset().validate(data).isOK()) { return; }
            api({
              type: 'POST',
              url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores',
              data: data,
              success: function () {
                goto("#!/stores");
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
        $('#main form').pluginForm({ type: 'store' });
      }
    });
    break; /* #!/stores */
    // }}}
  case '#!/stores/edit': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    apis({
      base: '/v2/tenants/'+$global.auth.tenant.uuid,
      multiplex: {
        store:  { type: 'GET', url: '+/stores/'+args.uuid },
        agents: { type: 'GET', url: '+/agents' },
      },
      error: "Failed to retrieve storage system information from the SHIELD API.",
      success: function (data) {
        $('#main').html($(template('stores-form', data))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializePluginObject();
            if (!$form.reset().validate(data).isOK()) { return; }
            api({
              type: 'PUT',
              url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores/'+args.uuid,
              data: data,
              success: function () {
                goto("#!/stores/store:uuid:"+args.uuid);
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
        $('#main form').pluginForm({
          type   : 'store',
          plugin : data.store.plugin,
          config : data.store.config
        });
        $('#main select[name="agent"]').trigger('change');
        $('#main select[name="plugin"]').trigger('change');
      }
    });

    break; /* #!/stores/edit */
    // }}}
  case '#!/stores/delete': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores/'+args.uuid,
      error: "Failed to retrieve storage system information from the SHIELD API.",
      success: function (store) {
        modal($(template('stores-delete', { store: store }))
          .on('click', '[rel="yes"]', function (event) {
            event.preventDefault();
            api({
              type: 'DELETE',
              url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/stores/'+args.uuid,
              error: "Unable to delete storage system",
              complete: function () {
                modal(true);
              },
              success: function (event) {
                goto('#!/stores');
              }
            });
          })
          .on('click', '[rel="close"]', function (event) {
            modal(true);
            goto('#!/stores/store:uuid:'+args.uuid);
          })
        );
      }
    });

    break; /* #!/stores/delete */
    // }}}

  case '#!/policies': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies',
      error: 'Failed to retrieve retention policy information from the SHIELD API.',
      success: function (data) {
        /* FIXME: data fixups that should probably migrate back to API */
        for (var i = 0; i < data.length; i++) {
          /* convert seconds -> days */
          data[i].days = data[i].expires / 86400;
        }

        $('#main').html(template('policies', { policies: data }));
      }
    });
    break; /* #!/policies */
    // }}}
  case '#!/policies/new': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    $('#main').html($(template('policies-form', { policy: null }))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        var data = $form.serializeObject();

        $form.reset();
        if (!parseInt(data.days) || parseInt(data.days) < 1) {
          $form.error('expires', 'invalid');
        }
        if (!$form.isOK()) {
          return;
        }

        data.expires = data.days * 86400; /* FIXME fixup for API */
        delete data.days;

        api({
          type: 'POST',
          url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies',
          data: data,
          success: function () {
            goto("#!/policies");
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      }));

    break; /* #!/policies/new */
    // }}}
  case '#!/policies/edit': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies/'+args.uuid,
      error: "Failed to retrieve retention policy information from the SHIELD API.",
      success: function (data) {
        data.days = parseInt(data.expires / 86400); /* FIXME fix this in API */
        $('#main').html($(template('policies-form', { policy: data }))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializeObject();

            $form.reset();
            if (!parseInt(data.days) || parseInt(data.days) < 0) {
              $form.error('expires', 'invalid');
            }
            if (!$form.isOK()) {
              return;
            }

            data.expires = data.days * 86400; /* FIXME fixup for API */
            delete data.days;

            api({
              type: 'PUT',
              url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies/'+args.uuid,
              data: data,
              success: function () {
                goto("#!/policies");
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
      }
    });

    break; /* #!/policies/edit */
    // }}}
  case '#!/policies/delete': /* {{{ */
    if (!$global.auth.tenant) {
      $('#main').html(template('you-have-no-tenants'));
      break;
    }
    if (!$global.auth.is.tenant[$global.auth.tenant.uuid].engineer) {
      $('#main').html(template('access-denied', { level: 'tenant', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies/'+args.uuid,
      error: "Failed to retrieve retention policy information from the SHIELD API.",
      success: function (data) {
        modal($(template('policies-delete', { policy: data }))
          .on('click', '[rel="yes"]', function (event) {
            event.preventDefault();
            api({
              type: 'DELETE',
              url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/policies/'+args.uuid,
              error: "Unable to delete retention policy",
              complete: function () {
                modal(true);
              },
              success: function (event) {
                goto('#!/policies');
              }
            });
          })
          .on('click', '[rel="close"]', function (event) {
            modal(true);
            goto('#!/policies');
          })
        );
      }
    });

    break; /* #!/admin/policies/delete */
    // }}}
  case '#!/tenants/edit': /* {{{ */
    if (!$global.auth.tenant) {
        $('#main').html(template('you-have-no-tenants'));
        break;
    }
    if (!$global.auth.is.tenant[args.uuid].admin) {
        $('#main').html(template('access-denied', { level: 'tenant', need: 'admin' }));
        break;
    }
    api({
      type: 'GET',
      url:  '/v2/tenants/'+args.uuid,
      error: "Failed to retrieve tenant information from the SHIELD API.",
      success: function (data) {
        var members = {};
        $.each(data.members, function (i, user) {
          members[user.uuid] = user;
        });
        $('#main').html($(template('tenants-form', { tenant: data, admin: false }))
          .userlookup('input[name=invite]', {
            filter: function (users) {
              var lst = [];
              $.each(users, function (i, user) {
                if (!(user.uuid in members)) {
                  lst.push(user);
                }
              });
              return lst;
            },
            onclick: function (user) {
              user.role = 'operator';
              $('#main table tbody').append(template('tenants-form-invitee', { user: user }));
              members[user.uuid] = user;

              api({
                type: 'POST',
                url:  '/v2/tenants/'+args.uuid+'/invite',
                data: {users:[user]},
                error: "Unable to save tenant role assignment.",
                success: function () {
                  banner('User "'+user.account+'" is now '+{
                      admin    : 'an administrator',
                      engineer : 'an engineer',
                      operator : 'an operator'
                    }[user.role]+' on this tenant.');
                }
              });
            }
          })
          .roles('.role', function (e, role) {
            var data = {
              uuid    : e.extract('uuid'),
              account : e.extract('account'),
              role    : e.extract('role')
            };
            api({
              type: 'POST',
              url:  '/v2/tenants/'+args.uuid+'/invite',
              data: {users:[data]},
              error: "Unable to save tenant role assignment.",
              success: function () {
                banner('User "'+data.account+'" is now '+{
                    admin    : 'an administrator',
                    engineer : 'an engineer',
                    operator : 'an operator'
                  }[data.role]+' on this tenant.');
              }
            });
          })
          .autofocus()
          .on('click', 'a[href="banish:user"]', function (event) {
            event.preventDefault();

            var e = $(event.target);
            var data = {
              uuid    : e.extract('uuid'),
              account : e.extract('account')
            };
            delete members[data.uuid];
            api({
              type: 'POST',
              url:  '/v2/tenants/'+args.uuid+'/banish',
              data: {users:[data]},
              error: "Unable to save tenant role assignment.",
              success: function () {
                banner('User "'+data.account+'" is no longer associated with this tenant.');
              }
            })
            $(event.target).closest('tr').remove();
          }));
      }
    });

    break; /* #!/tenants/edit */
    // }}}

  case '#!/admin': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('admin'));
    break; /* #!/admin */
    // }}}
  case '#!/admin/agents': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/agents',
      error: "Failed retrieving the list of agents from the SHIELD API.",
      success: function (data) {
        $('#main').html($(template('agents', data))
          .on('click', 'a[rel]', function (event) {
            var action = $(event.target).closest('a[rel]').attr('rel');
            if (action == 'hide' || action == 'show') {
              event.preventDefault();
              api({
                type: 'POST',
                url:  '/v2/agents/'+$(event.target).extract('agent-uuid')+'/'+action,
                error: "Unable to "+action+" agent via the SHIELD API.",
                success: function () { reload(); }
              });
            }
          }));
      }
    });
    break; /* #!/admin/agents */
    // }}}
  case '#!/admin/auth': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/auth/providers',
      error: "Failed retrieving the list of configured authentication providers from the SHIELD API.",
      success: function (data) {
        $('#main').html(template('auth-providers', { providers: data }));
      }
    });
    break; /* #!/admin/auth */
    // }}}
  case '#!/admin/auth/config': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/auth/providers/'+args.name,
      error: "Failed retrieving the authentication provider configuration from the SHIELD API.",
      success: function (data) {
        $('#main').html(template('auth-provider-config', { provider: data }));
      }
    });
    break; /* #!/admin/auth */
    // }}}
  case '#!/admin/rekey': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html($(template('rekey')))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        var data = $form.serializeObject();

        $form.reset();
        if (data.current == "") {
          $form.error('current', 'missing');
        }

        if (data.new == "") {
          $form.error('new', 'missing');

        } else if (data.confirm == "") {
          $form.error('confirm', 'missing');

        } else if (data.new != data.confirm) {
          $form.error('confirm', 'mismatch');
        }

        if (!$form.isOK()) {
          return;
        }

        delete data.confirm;
        api({
          type: 'POST',
          url:  '/v2/rekey',
          data: data,
          success: function () {
            banner('Succcessfully rekeyed the SHIELD Core.');
            goto("#!/admin");
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      });

    break; /* #!/admin/rekey */
    // }}}

  case '#!/admin/tenants': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/tenants',
      error: 'Failed to retrieve tenant information from the SHIELD API.',
      success: function (data) {
        $('#main').html(template('tenants', { tenants: data, admin: true }));
      }
    });
    break; /* #!/admin/tenants */
    // }}}
  case '#!/admin/tenants/new': /* {{{ */
    if (!$global.auth.is.system.manager) {
      $('#main').html(template('access-denied', { level: 'system', need: 'manager' }));
      break;
    }
    var members = {};

    $('#main').html($(template('tenants-form', { policy: null, admin: true }))
      .userlookup('input[name=invite]', {
        // {{{
        filter: function (users) {
          var lst = [];
          $.each(users, function (i, user) {
            if (!(user.uuid in members)) {
              lst.push(user);
            }
          });
          return lst;
        },
        onclick: function (user) {
          user.role = 'operator';
          $('#main table tbody').append(template('tenants-form-invitee', { user: user }));
          members[user.uuid] = user.role;
        }
        // }}}
      })
      .roles('.role', function (e, role) {
        members[e.extract('uuid')] = role;
      })
      .autofocus()
      .on('click', 'a[href="banish:user"]', function (event) {
        // {{{
        event.preventDefault();
        delete members[$(event.target).extract('uuid')];
        $(event.target).closest('tr').remove();
        // }}}
      })
      .on('submit', 'form', function (event) {
        // {{{
        event.preventDefault();

        var $form = $(event.target);
        var data = $form.serializeObject();
        data.users = [];
        for (uuid in members) {
          data.users.push({
            uuid: uuid,
            role: members[uuid]
          });
        }

        $form.reset();

        api({
          type: 'POST',
          url:  '/v2/tenants',
          data: data,
          success: function () {
            goto("#!/admin/tenants");
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
        // }}}
      }));

    break; /* #!/admin/tenants/new */
    // }}}
  case '#!/admin/tenants/edit': /* {{{ */
    if (!$global.auth.is.system.manager) {
      $('#main').html(template('access-denied', { level: 'system', need: 'manager' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/tenants/'+args.uuid,
      error: "Failed to retrieve tenant information from the SHIELD API.",
      success: function (data) {
        var members = {};
        $.each(data.members, function (i, user) {
          members[user.uuid] = user;
        });
        $('#main').html($(template('tenants-form', { tenant: data, admin: true }))
          .userlookup('input[name=invite]', {
            filter: function (users) {
              var lst = [];
              $.each(users, function (i, user) {
                if (!(user.uuid in members)) {
                  lst.push(user);
                }
              });
              return lst;
            },
            onclick: function (user) {
              user.role = 'operator';
              $('#main table tbody').append(template('tenants-form-invitee', { user: user }));
              members[user.uuid] = user;

              api({
                type: 'POST',
                url:  '/v2/tenants/'+args.uuid+'/invite',
                data: {users:[user]},
                error: "Unable to save tenant role assignment.",
                success: function () {
                  banner('User "'+user.account+'" is now '+{
                      admin    : 'an administrator',
                      engineer : 'an engineer',
                      operator : 'an operator'
                    }[user.role]+' on this tenant.');
                }
              });
            }
          })
          .roles('.role', function (e, role) {
            var data = {
              uuid    : e.extract('uuid'),
              account : e.extract('account'),
              role    : e.extract('role')
            };
            api({
              type: 'POST',
              url:  '/v2/tenants/'+args.uuid+'/invite',
              data: {users:[data]},
              error: "Unable to save tenant role assignment.",
              success: function () {
                banner('User "'+data.account+'" is now '+{
                    admin    : 'an administrator',
                    engineer : 'an engineer',
                    operator : 'an operator'
                  }[data.role]+' on this tenant.');
              }
            });
          })
          .autofocus()
          .on('click', 'a[href="banish:user"]', function (event) {
            event.preventDefault();

            var e = $(event.target);
            var data = {
              uuid    : e.extract('uuid'),
              account : e.extract('account')
            };
            delete members[data.uuid];
            api({
              type: 'POST',
              url:  '/v2/tenants/'+args.uuid+'/banish',
              data: {users:[data]},
              error: "Unable to save tenant role assignment.",
              success: function () {
                banner('User "'+data.account+'" is no longer associated with this tenant.');
              }
            })
            $(event.target).closest('tr').remove();
          })
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializeObject();

            $form.reset();

            api({
              type: 'PATCH',
              url:  '/v2/tenants/'+args.uuid,
              data: data,
              success: function () {
                goto("#!/admin/tenants");
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
      }
    });

    break; /* #!/admin/tenants/edit */
    // }}}

  case '#!/admin/users': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/auth/local/users',
      error: "Failed retrieving the list of local SHIELD users from the SHIELD API.",
      success: function (data) {
        $('#main').html(template('admin-users', { users: data }));
      }
    });
    break; /* #!/admin/users */
    // }}}
  case "#!/admin/users/new": /* {{{ */
    if (!$global.auth.is.system.manager) {
      $('#main').html(template('access-denied', { level: 'system', need: 'manager' }));
      break;
    }
    $('#main').html($(template('admin-users-new', {})))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();
        var $form = $(event.target);

        var payload = {
          name:     $form.find('[name=name]').val(),
          account:  $form.find('[name=account]').val(),
          password: $form.find('[name=password]').val()
        };

        if ($form.find('[name=confirm]').val() != payload.password) {
          banner("Passwords don't match", "error");
          return;
        }

        banner("Creating new user...", "info");
        api({
          type: 'POST',
          url:  '/v2/auth/local/users',
          data: payload,
          success: function (data) {
            banner('New user created successfully.');
            goto("#!/admin/users");
          },
          error: function (xhr) {
            banner("Failed to create new user", "error");
          }
        });
      });
    break; // #!/admin/users/new
    // }}}

  case '#!/admin/stores': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/global/stores',
      error: "Failed retrieving the list of storage endpoints from the SHIELD API.",
      success: function (data) {
        /* FIXME fixups that need to migrate into the SHIELD code */
        for (key in data) {
          if (!('ok' in data[key])) {
            data[key].ok = true;
          }
        }
        $('#main').html(template('stores', { stores: data, admin: true }));
      }
    });
    break; /* #!/admin/stores */
    // }}}
  case '#!/admin/stores/store': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/global/stores/'+args.uuid,
      error: "Unable to retrieve storage systems from SHIELD API.",
      success: function (data) {
        /* FIXME fixups that need to migrate into the SHIELD code */
        data.ok = true;
        data.archives = data.archive_count;
        data.used = data.storage_used;
        data.threshold = data.threshold;
        data.projected = 2.1;
        data.daily_delta = data.daily_increase;
        $('#main').html(template('store', { store: data, admin: true }));
      }
    });
    break; /* #!/admin/stores/store */
    // }}}
  case '#!/admin/stores/new': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/agents',
      error: "Unable to retrieve list of SHIELD Agents from the SHIELD API",
      success: function (data) {
        var cache = {};

        $('#main').html($(template('stores-form', {
            agents: data.agents,
            admin:  true,
          }))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializePluginObject();
            if (!$form.reset().validate(data).isOK()) { return; }
            api({
              type: 'POST',
              url:  '/v2/global/stores',
              data: data,
              success: function () {
                goto("#!/admin/stores");
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
        $('#main form').pluginForm({ type: 'store' });
      }
    });

    break; /* #!/admin/stores */
    // }}}
  case '#!/admin/stores/edit': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    apis({
      multiplex: {
        store:  { type: 'GET', url: '/v2/global/stores/'+args.uuid },
        agents: { type: 'GET', url: '/v2/agents' }
      },
      error: "Failed to retrieve storage system information from the SHIELD API.",
      success: function (data) {
        data.admin = true;
        data.agents = data.agents.agents;
        $('#main').html($(template('stores-form', data))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializePluginObject();
            if (!$form.reset().validate(data).isOK()) { return; }
            api({
              type: 'PUT',
              url:  '/v2/global/stores/'+args.uuid,
              data: data,
              success: function () {
                goto("#!/admin/stores/store:uuid:"+args.uuid);
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
        $('#main form').pluginForm({
          type   : 'store',
          plugin : data.store.plugin,
          config : data.store.config
        });
        $('#main select[name="agent"]').trigger('change');
        $('#main select[name="plugin"]').trigger('change');
      }
    });

    break; /* #!/admin/stores/edit */
    // }}}
  case '#!/admin/stores/delete': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/global/stores/'+args.uuid,
      error: "Failed to retrieve storage system information from the SHIELD API.",
      success: function (store) {
        modal($(template('stores-delete', { store: store }))
          .on('click', '[rel="yes"]', function (event) {
            event.preventDefault();
            api({
              type: 'DELETE',
              url:  '/v2/global/stores/'+args.uuid,
              error: "Unable to delete storage system",
              complete: function () {
                modal(true);
              },
              success: function (event) {
                goto('#!/admin/stores');
              }
            });
          })
          .on('click', '[rel="close"]', function (event) {
            modal(true);
            goto('#!/admin/stores/store:uuid:'+args.uuid);
          })
        );
      }
    });

    break; /* #!/admin/stores/delete */
    // }}}

  case '#!/admin/policies': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/global/policies',
      error: "Failed retrieving the list of retention policy templates from the SHIELD API.",
      success: function (data) {
        /* FIXME: data fixups that should probably migrate back to API */
        for (var i = 0; i < data.length; i++) {
          /* convert seconds -> days */
          data[i].days = data[i].expires / 86400;
        }

        $('#main').html(template('policies', { policies: data, admin: true }));
      }
    });
    break; /* #!/admin/policies */
    // }}}
  case '#!/admin/policies/new': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html($(template('policies-form', { policy: null, admin: true }))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        var data = $form.serializeObject();

        $form.reset();
        if (!parseInt(data.days) || parseInt(data.days) < 1) {
          $form.error('expires', 'invalid');
        }
        if (!$form.isOK()) {
          return;
        }

        data.expires = data.days * 86400; /* FIXME fixup for API */
        delete data.days;

        api({
          type: 'POST',
          url:  '/v2/global/policies',
          data: data,
          success: function () {
            goto("#!/admin/policies");
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      }));
    break; /* #!/admin/policies/new */
    // }}}
  case '#!/admin/policies/edit': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/global/policies/'+args.uuid,
      error: "Failed to retrieve retention policy template information from the SHIELD API.",
      success: function (data) {
        data.days = parseInt(data.expires / 86400); /* FIXME fix this in API */
        $('#main').html($(template('policies-form', { policy: data, admin: true }))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();

            var $form = $(event.target);
            var data = $form.serializeObject();

            $form.reset();
            if (!parseInt(data.days) || parseInt(data.days) < 1) {
              $form.error('expires', 'invalid');
            }
            if (!$form.isOK()) {
              return;
            }

            data.expires = data.days * 86400; /* FIXME fixup for API */
            delete data.days;

            api({
              type: 'PUT',
              url:  '/v2/global/policies/'+args.uuid,
              data: data,
              success: function () {
                goto("#!/admin/policies");
              },
              error: function (xhr) {
                $form.error(xhr.responseJSON);
              }
            });
          }));
        }
      });
      break; /* #!/admin/policies/edit */
    // }}}
  case '#!/admin/policies/delete': /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/global/policies/'+args.uuid,
      error: "Failed to retrieve retention policy template information from the SHIELD API.",
      success: function (data) {
        modal($(template('policies-delete', { policy: data, admin: true }))
          .on('click', '[rel="yes"]', function (event) {
            event.preventDefault();
            api({
              type: 'DELETE',
              url:  '/v2/global/policies/'+args.uuid,
              error: "Unable to delete retention policy template",
              complete: function () {
                modal(true);
              },
              success: function (event) {
                goto('#!/admin/policies');
              }
            });
          })
          .on('click', '[rel="close"]', function (event) {
            modal(true);
            goto('#!/admin/policies');
          })
        );
      }
    });

    break; /* #!/admin/policies/delete */
    // }}}
  case '#!/admin/sessions': /* {{{ */
    if (!$global.auth.is.system.admin) {
      $('#main').html(template('access-denied', { level: 'system', need: 'admin' }));
      break;
    }
    $('#main').html(template('loading'));
    api({
      type: 'GET',
      url:  '/v2/auth/sessions',
      error: "Failed retrieving the list of sessions from the SHIELD API.",
      success: function (data) {
      data = data.sort(function(a, b) {
        if (a.user_account != b.user_account){
            return a.user_account > b.user_account;
        }
        return tparse(a.last_seen_at).getTime() < tparse(b.last_seen_at).getTime();
      });
      $('#main').html(template('sessions', { sessions: data, admin: true }));
      }
    });
    break; /* #!/admin/sessions */
    // }}}
  case '#!/admin/sessions/delete': /* {{{ */
    if (!$global.auth.is.system.admin) {
      $('#main').html(template('access-denied', { level: 'system', need: 'admin' }));
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/auth/sessions/'+args.uuid,
      error: "Failed to retrieve session information from the SHIELD API.",
      success: function (data) {
      modal($(template('sessions-delete', { session: data }))
        .on('click', '[rel="yes"]', function (event) {
        event.preventDefault();
        api({
            type: 'DELETE',
            url:  '/v2/auth/sessions/'+args.uuid,
            error: "Unable to delete session",
            complete: function () {
            modal(true);
            },
            success: function (event) {
            goto('#!/admin/sessions');
            }
        });
        })
        .on('click', '[rel="close"]', function (event) {
        modal(true);
        goto('#!/admin/sessions');
        })
    );
    }
    });
    break; /* #!/admin/sessions/delete */
    // }}}
  case "#!/unlock": /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html($(template('unlock', {})))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var data = $(event.target).serializeObject();

        api({
          type: 'POST',
          url:  '/v2/unlock',
          data: data,
          error: "Unable to unlock the SHIELD Core.",
          success: function (data) {
            $global.hud.health.core = "unlocked";
            $('#hud').html(template('hud', $global.hud));
            goto("");
          }
        });
      });
    break;
    // }}}
  case "#!/init": /* {{{ */
    if (!$global.auth.is.system.engineer) {
      $('#main').html(template('access-denied', { level: 'system', need: 'engineer' }));
      break;
    }
    $('#main').html($(template('init', {})))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        var data = $form.serializeObject();

        $form.reset();
        if (data.master == "") {
          $form.error('master', 'missing');

        } else if (data.confirm == "") {
          $form.error('confirm', 'missing');

        } else if (data.master != data.confirm) {
          $form.error('confirm', 'mismatch');
        }

        if (!$form.isOK()) {
          return;
        }

        api({
          type: 'POST',
          url:  '/v2/init',
          data: { "master": data.master },
          success: function (data) {
            $global.hud.health.core = "unlocked";
            $('#hud').html(template('hud', $global.hud));
            goto("");
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      });
    break;
    // }}}

  default: /* 404 {{{ */
    $('#main').html(template('404', {
      wanted:  page,
      args:    argv,
      referer: referer,
    }));
    return; /* 404 */
    // }}}
  }
  referer = page;
}

function redraw(complete) {
  if (complete && !$global.auth.unauthenticated) {
    $('#viewport').html(template('layout'));
  }
  $('#hud').html(template('hud', $global.hud));
  $('.top-bar').html(template('top-bar', {
    shield:  $global.hud.shield,
    user:    $global.auth.user,
    tenants: $global.auth.tenants,
    tenant:  $global.auth.tenant
  }));
}
function goto(page) {
  if (document.location.hash == page) {
    dispatch(page); // re-dispatch
  } else {
    document.location.hash = page;
  }
}
function reload() {
  goto(document.location.hash)
}

$(function () {
  if (!$global.auth.unauthenticated) {
    $('#viewport').html(template('layout'));
  }

  /* ... watch the document hash for changes {{{ */
  $(window).on('hashchange', function (event) {
    dispatch(document.location.hash);
  }).trigger('hashchange');
  /* }}} */
  /* ... ping /v2/health every X seconds, and update the HUD {{{ */
  (function (s) {
    var last    = "",
        every   = 5,
        timer   = undefined,
        failing = true,
        backoff = {
          0:   5,    /* on success */
          5:   6,    /* +1  */
          6:   7,    /* +1  */
          7:   9,    /* +2  */
          9:  11,    /* +3  */
          11: 16,    /* +5  */
          16: 24,    /* +8  */
          24: 37,    /* +13 */
          37: 60,    /* +21 (round to 60) */
          60: 60     /* max out at 1min */
        };

    var rehud = function (data) {
      $global.hud = data;
      var json = JSON.stringify(data);
      if (json != last) { redraw(false); }
      last = json;
    };
    var ping = function () {
      var uri = ""
      if ($global.auth.tenant) {
        uri = '/v2/tenants/'+$global.auth.tenant.uuid+'/health'
      } else {
        uri = '/v2/health'
      }
      $.ajax({
        type: 'GET',
        url:  uri,
        success: function (data) {
          if (failing) {
            failing = false;
            every = backoff[0]; /* reset */
          } else {
            every = backoff[every];
          }
          rehud(data);
        },
        error: function (xhr) {
          failing = true;
          var old = every;
          every = backoff[every];
          if (every != backoff[every]) {
            console.log('/v2/health check failed; backing off to check every %d seconds', every);
          }

          $global.hud.health.core = 'unreachable';
          rehud($global.hud);
        },
        complete: function () {
          timer = window.setTimeout(ping, every * 1000);
        },
        statusCode: {
          401: function() {
            console.log('/v2/health check received a 401, redirecting to login page')
            api({
              type: "GET",
              url: "/v2/auth/logout",
              success: function () {
                document.location.href = '/';
              },
              error: function (xhr) {
                if (xhr.status >= 500) {
                  $('#viewport').html(template('BOOM'));
                } else {
                  document.location.href = '/';
                }
              }
            })
          },
          403: function() {
            console.log('/v2/health check received a 403, redirecting to login page')
            api({
              type: "GET",
              url: "/v2/auth/logout",
              success: function () {
                document.location.href = '/';
              },
              error: function (xhr) {
                if (xhr.status >= 500) {
                  $('#viewport').html(template('BOOM'));
                } else {
                  document.location.href = '/';
                }
              }
            })
          },
        }
      });
    }
    $(document).on('visibilitychange', function (event) {
      if (document.hidden) {
        console.log('pausing /v2/health checks...');
        if (timer) { window.clearTimeout(timer); }
        timer = undefined;
      } else {
        console.log('resuming /v2/health checks...');
        every = backoff[0]; /* reset */
        //dont ping health if you're on the login page
        if ($global.auth.unauthenticated){return}
        ping();
      }
    });
    //dont ping health if you're on the login page
    if ($global.auth.unauthenticated){return}
    ping();
  })(5);
  /* }}} */
  /* ... handle the account menu {{{ */
  $(document.body).on('click', '.top-bar a[rel=account]', function (event) {
    event.preventDefault();
    event.stopPropagation();
    $('.top-bar .flyout').toggle();
  });
  $(document.body).on('click', '.top-bar a[href^="switchto:"]', function (event) {
    event.preventDefault();
    var uuid = $(event.target).attr('href').replace(/^switchto:/, '');
    for (var i = 0; i < $global.auth.tenants.length; i++) {
      if ($global.auth.tenants[i].uuid == uuid) {
        $global.auth.tenant = $global.auth.tenants[i];
        api({
          type: 'PATCH',
          url:  '/v2/auth/user/settings',
          data: { default_tenant: uuid }
        });

        redraw(true);
        var page = document.location.hash.replace(/^(#!\/[^\/]*).*/, '$1');
        if (page == "#!/do") { page = "#!/systems"; }
        if (page == "#!/tenants") { page = "#!/systems"; }
        goto(page);
        return;
      }
    }
  });
  $(document.body).on('click', '.top-bar .fly-out', function (event) {
    event.preventDefault();
    event.stopPropagation();
  });
  $(document.body).on('click', function (event) {
    $('.ephemeral').hide();
  });
  /* }}} */

  /* global: show a task in the next row down {{{ */
  $(document.body).on('click', 'a[href^="task:"]', function (event) {
    event.preventDefault();
    var uuid  = $(event.target).closest('a[href^="task:"]').attr('href').replace(/^task:/, '');
    var $ev   = $(event.target).closest('.event');
    var $task = $ev.find('.task');

    if ($task.is(':visible')) {
      $task.hide();
      return;
    }

    $task = $task.show()
                .html(template('loading'));

    api({
      type: 'GET',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/tasks/'+uuid,
      error: "Failed to retrieve task information from the SHIELD API.",
      success: function (data) {
        $task.html(template('task', {
          task: data,
          restorable: data.archive_uuid != "" && data.status == "done",
        }));
      }
    });
  });
  /* }}} */
  /* global: show an annotation form, in a task log {{{ */
  $(document.body).on('click', '.task button[rel^="annotate:"]', function (event) {
    $(event.target).closest('.task').find('form.annotate').toggle();
  });
  /* }}} */
  /* global: submit the annotation form {{{ */
  $(document.body).on('submit', '.task form.annotate', function (event) {
    event.preventDefault();

    var $form = $(event.target);
    var uuid = $form.extract('system-uuid');

    if ($form.is('[data-system-uuid] [data-task-uuid] *')) {
      var ann = {
        type  : "task",
        uuid  : $form.find('[name=uuid]').val(),
        notes : $form.find('[name=notes]').val()
      };
      if ($form.find('input[name=disposition]').length > 0) {
        ann.disposition = $form.find('input[name=disposition]').is(':checked')
                        ? "ok" : "failed";
      }
      ann.clear = $form.find('[optgroup=clear]:checked').val();
      if (ann.clear == '') {
        ann.clear = "normal";
      }

      api({
        type: 'PATCH',
        url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/systems/'+uuid,
        data: { "annotations": [ann] },
        success: function (data) {
          $form.hide();
          banner("task annotation saved.");
          reload();
        },
        error: function (xhr) {
          $form.hide();
          banner("task annotation failed to save.", 'error');
        }
      });

    } else {
      throw 'unexpected annotation form (not a .tasks or .archives descendent)'
    }
  });
  /* }}} */

  /* global: handle "run:job-uuid" links {{{ */
  $(document.body).on('click', 'a[href^="run:"], button[rel^="run:"]', function (event) {
    event.preventDefault();
    var uuid;
    if ($(event.target).is('button')) {
      uuid = $(event.target).attr('rel');
    } else {
      uuid  = $(event.target).closest('a[href^="run:"]').attr('href');
    }
    uuid = uuid.replace(/^run:/, '');

    banner('scheduling ad hoc backup...', 'progress');
    api({
      type: 'POST',
      url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/jobs/'+uuid+'/run',
      success: function () {
        banner('ad hoc backup job scheduled');
        reload();
      },
      error: function () {
        banner('unable to schedule ad hoc backup job', 'error');
      }
    });
  });
  /* }}} */
  /* global: handle "restore:archive-uuid" buttons {{{ */
  $(document.body).on('click', '.task button[rel^="restore:"]', function (event) {
    var uuid   = $(event.target).extract('archive-uuid');
    var target = $(event.target).extract('system-name');
    var taken  = $(event.target).extract('archive-taken');
    console.log('restoring archive %s!', uuid);

    modal(template('restore-are-you-sure', {
        target: target,
        taken:  taken
      })).on('click', '[rel=yes]', function(event) {
      event.preventDefault();
      api({
        type: 'POST',
        url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/archives/'+uuid+'/restore',
        success: function() {
          banner("restore operation started");
        },
        error: function () {
          banner("unable to schedule restore operation", "error");
        }
      });
    });
  });
  /* }}} */
});
