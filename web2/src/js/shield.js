// vim:et:sts=2:ts=2:sw=2
var SHIELD, referer;

function divert(page) { // {{{
  if (page.match(/^#!\/(login|logout|cliauth)$/)) {
    /* never divert these pages */
    return page;
  }

  if (!SHIELD.authenticated()) {
    console.log('session not authenticated; diverting to #!/login page...');
    return "#!/login";
  }

  if (SHIELD.is('engineer') && SHIELD.shield) {
    /* process 'system' team diverts */
    if (SHIELD.shield.core == "uninitialized") {
      console.log('system user detected, and this SHIELD core is uninitialized; diverting to #!/init page...');
      return "#!/init";

    } else if (SHIELD.shield.core == "sealed" || SHIELD.shield.core == "locked") {
      console.log('system user detected, and this SHIELD core is locked; diverting to #!/unlock page...');
      return "#!/unlock";
    }
  }

  if (!page || page == "") {
    return SHIELD.is('engineer') ? '#!/admin' : '#!/systems';
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
        $('#viewport').append($.template('logging-in', {auth: how}));
      }

      api({
        type: 'GET',
        url:  '/v2/auth/providers?for=web',
        success: function (data) {
          $('#viewport').html($($.template('login', { providers: data }))
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
            }));
        },
        error: function (xhr) {
          $('#viewport').template('BOOM');
        }
      });
    })();
    break; /* #!/login */
    // }}}
  case "#!/cliauth": /* {{{ */
    $('#viewport').template('cliauth', args);
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
            $('#viewport').template('BOOM');
          } else {
            document.location.href = '/';
          }
        }
      });
    })()
    break;
    // }}}

    case "#!/init": /* {{{ */
      (function () {
        $('#viewport').template('init');
        $('#viewport').html($($.template('init'))
          .on("submit", ".restore", function (event) {
            event.preventDefault();
           // progress('Initializing SHIELD with prior backup');

            var $form = $(event.target);
            var data = new FormData();

            if ($form[0].fixedkey.value.length < 512 || $form[0].fixedkey.value.length > 512) {
              $form.error('fixedkey', 'missing')
              return;
            }
            data.append("archive", $form[0].archive.files[0]);
            data.append("fixedkey", $form[0].fixedkey.value);

            $form.reset();
            $('.dialog').html("")
            $('.dialog').template('loading')
            $('.dialog').prepend("<h2 style=\"text-align: center;\">SHIELD is initializing from a previous backup, please wait...</h2>")

            $.ajax({
              type: "POST",
              url: "/v2/bootstrap/restore",
              data: data,
              cache: false,
              contentType: false,
              processData: false,
              success: function () {
                $('.dialog').template('loading')
                $('.dialog').prepend("<h2 style=\"text-align: center;\">SHIELD initialization success, taking you authentication...</h2>")
              },
              error: function () {
                $('.dialog').template('loading')
                $('.dialog').prepend("<h2 style=\"text-align: center;\">SHIELD initialization failed, restarting initialization process...</h2>")
              }
            });
          })
          .on("submit", ".setpass", function (event) {
            event.preventDefault();
            var $form = $(event.target);
            var data = $form.serializeObject();
            if (data.masterpass == "") {
              $form.error('masterpass', 'missing');

            } else if (data.masterpassconf == "") {
              $form.error('masterpassconf', 'missing');

            } else if (data.masterpass != data.masterpassconf) {
              $form.error('masterpassconf', 'mismatch');
            }

            if (!$form.isOK()) {
              return;
            }
            api({
              type: 'POST',
              url: '/v2/init',
              data: { "master": data.masterpass },
              success: function (data) {
                console.log("success");
                $('#viewport').template('fixedkey', data);
              },
              error: function (xhr) {
                $(event.target).error(xhr.responseJSON);
              }
            });
          })
        );
        $.ajax({
          type: "GET",
          url: "/v2/bootstrap/log",
          success: function (data) {
            if (data["task"]["log"] != "") {
              $('.restore_divert').html("It looks like there was a previous attempt to self-restore SHIELD that failed. Below is the task log to help debug the problem. ")
              $('#initialize').append("<div class=\"dialog\" id=\"log\"></div>")
              $('#log').append($.template('task', data))
            }
          }
        });
      })();
      break; /* #!/init */
    // }}}

  case "#!/do/backup": /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    if (!SHIELD.is('operator', SHIELD.activeTenant())) {
      $('#main').template('access-denied', { level: 'tenant', need: 'operator' });
      break;
    }
    $('#main').template('do-backup');
    new DoAdHocWizard($('#main'), '.do-backup');
    break; /* #!/do/backup */
    // }}}
  case "#!/do/restore": /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    if (!SHIELD.is('operator', SHIELD.activeTenant())) {
      $('#main').template('access-denied', { level: 'tenant', need: 'operator' });
      break;
    }

    $('#main').template('do-restore');
    new DoRestoreWizard($('#main'), '.do-restore');
    break; /* #!/do/restore */
    // }}}
  case "#!/do/configure": /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    if (!SHIELD.is('engineer', SHIELD.activeTenant())) {
      $('#main').template('access-denied', { level: 'tenant', need: 'engineer' });
      break;
    }

    var data = {};
    $('#main').template('do-configure', data);
    $(document.body)
      .on('change', '#main select[name="target.plugin"]', function (event) {
        data.selected_target_plugin = $(event.target).val();
        $('#main .redraw.target').template('do-configure-target-plugin', data)
                                .find('[name="target.agent"]').focus();
      })
      .on('change', '#main select[name="target.agent"]', function (event) {
        data.selected_target_agent = $(event.target).val();
        $('#main .redraw.target').template('do-configure-target-plugin', data)
                                 .find('.plugin0th input').focus();
      })
      .on('change', '#main select[name="store.plugin"]', function (event) {
        data.selected_store_plugin = $(event.target).val();
        $('#main .redraw.store').template('do-configure-store-plugin', data)
                                .find('[name="store.agent"]').focus();
      })
      .on('change', '#main select[name="store.agent"]', function (event) {
        data.selected_store_agent = $(event.target).val();
        $('#main .redraw.store').template('do-configure-store-plugin', data)
                                .find('.plugin0th input').focus();
      });
    window.setTimeout(function () {
      $('#main .optgroup').optgroup();

      new DoConfigureWizard($('#main'), '.do-configure');
      $('#main .scheduling [data-subform=schedule-daily]').trigger('click');
    }, 150);
    break; /* #!/do/configure */
    // }}}

  case "#!/systems": /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    $('#main').template('systems');
    break; /* #!/systems */
    // }}}
  case '#!/systems/system': /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    $('#main').template('loading');
    $('#main').template('system', { target: SHIELD.system(args.uuid) });
    window.setTimeout(function () {
      /* for some reason, we need a small delay before we trigger the load-more */
      $('#main .paginate .load-more').trigger('click');
    }, 210);
    break; /* #!/systems/system */
    // }}}

  case '#!/stores': /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    $('#main').template('stores');
    break; /* #!/stores */
    // }}}
  case '#!/stores/store': /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    $('#main').template('store', args);
    break; /* #!/stores/store */
    // }}}
  case '#!/stores/new': /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    if (!SHIELD.is('engineer', SHIELD.activeTenant())) {
      $('#main').template('access-denied', { level: 'tenant', need: 'engineer' });
      break;
    }
    $('#main').template('loading');
    var data = { type: 'store' };
    $('#main').html($($.template('stores-form', data))
      .autofocus()
      .on('change', 'select[name="store.plugin"]', function (event) {
        data.plugin = $(event.target).val();
        console.log(data);
        $('#main .redraw.store').template('plugin-form-agent-selector', data);
      })
      .on('change', 'select[name="store.agent"]', function (event) {
        data.agent = $(event.target).val();
        console.log(data);
        $('#main .redraw.store').template('plugin-form-agent-selector', data);
      })
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        if (!$form.reset().validate().isOK()) { return; }
        var data = $form.serializeObject().store;
        data.threshold = readableToBytes(data.threshold);
        api({
          type: 'POST',
          url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/stores',
          data: data,
          success: function () {
            goto("#!/stores");
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      }));
    break; /* #!/stores */
    // }}}
  case '#!/stores/edit': /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    if (!SHIELD.is('engineer', SHIELD.activeTenant())) {
      $('#main').template('access-denied', { level: 'tenant', need: 'engineer' });
      break;
    }
    $('#main').html($($.template('stores-form', {
        store: SHIELD.store(args.uuid)
      }))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        if (!$form.reset().validate().isOK()) { return; }
        var data = $form.serializeObject().store;
        data.threshold = readableToBytes(data.threshold);

        api({
          type: 'PUT',
          url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/stores/'+args.uuid,
          data: data,
          success: function () {
            goto("#!/stores/store:uuid:"+args.uuid);
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      }));

    break; /* #!/stores/edit */
    // }}}
  case '#!/stores/delete': /* {{{ */
    if (!SHIELD.activeTenant()) {
      $('#main').template('you-have-no-tenants');
      break;
    }
    if (!SHIELD.is('engineer', SHIELD.activeTenant())) {
      $('#main').template('access-denied', { level: 'tenant', need: 'engineer' });
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/tenants/'+SHIELD.activeTenant()+'/stores/'+args.uuid,
      error: "Failed to retrieve storage system information from the SHIELD API.",
      success: function (store) {
        modal($($.template('stores-delete', { store: store }))
          .on('click', '[rel="yes"]', function (event) {
            event.preventDefault();
            api({
              type: 'DELETE',
              url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/stores/'+args.uuid,
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

  case '#!/tenants/edit': /* {{{ */
    if (!SHIELD.activeTenant()) {
        $('#main').template('you-have-no-tenants');
        break;
    }
    if (!SHIELD.is('admin', args.uuid)) {
        $('#main').template('access-denied', { level: 'tenant', need: 'admin' });
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
        $('#main').html($($.template('tenants-form', { tenant: data, admin: false }))
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
              $('#main table tbody').append($.template('tenants-form-invitee', { user: user }));
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
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('admin');
    break; /* #!/admin */
    // }}}
  case '#!/admin/agents': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('loading');
    api({
      type: 'GET',
      url:  '/v2/agents',
      error: "Failed retrieving the list of agents from the SHIELD API.",
      success: function (data) {
        $('#main').html($($.template('agents', data))
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
            } else if (action == 'resync') {
              event.preventDefault();
              api({
                type: 'POST',
                url:  '/v2/agents/'+$(event.target).extract('agent-uuid')+'/resync',
                error: "Resynchronization request failed",
                success: function () {
                  banner("Resynchronization of agent underway");
                }
              });
            }
          }));
      }
    });
    break; /* #!/admin/agents */
    // }}}
  case '#!/admin/auth': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('loading');
    api({
      type: 'GET',
      url:  '/v2/auth/providers',
      error: "Failed retrieving the list of configured authentication providers from the SHIELD API.",
      success: function (data) {
        $('#main').template('auth-providers', { providers: data });
      }
    });
    break; /* #!/admin/auth */
    // }}}
  case '#!/admin/auth/config': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('loading');
    api({
      type: 'GET',
      url:  '/v2/auth/providers/'+args.name,
      error: "Failed retrieving the authentication provider configuration from the SHIELD API.",
      success: function (data) {
        $('#main').template('auth-provider-config', { provider: data });
      }
    });
    break; /* #!/admin/auth */
    // }}}
  case '#!/admin/rekey': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').html($($.template('rekey')))
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

        data.rotate_fixed_key = (data.rotate_fixed_key == "true");

        if (!$form.isOK()) {
          return;
        }

        delete data.confirm;
        api({
          type: 'POST',
          url:  '/v2/rekey',
          data: data,
          success: function (data) {
            if (data.fixed_key != "") {
              $('#viewport').template('fixedkey', data);
            } else {
              goto("#!/admin");
            }
            banner('Succcessfully rekeyed the SHIELD Core.');
          },
          error: function (xhr) {
            $form.error(xhr.responseJSON);
          }
        });
      });

    break; /* #!/admin/rekey */
    // }}}

  case '#!/admin/tenants': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('loading');
    api({
      type: 'GET',
      url:  '/v2/tenants',
      error: 'Failed to retrieve tenant information from the SHIELD API.',
      success: function (data) {
        $('#main').template('tenants', { tenants: data, admin: true });
      }
    });
    break; /* #!/admin/tenants */
    // }}}
  case '#!/admin/tenants/new': /* {{{ */
    if (!SHIELD.is('manager')) {
      $('#main').template('access-denied', { level: 'system', need: 'manager' });
      break;
    }
    var members = {};

    $('#main').html($($.template('tenants-form', { policy: null, admin: true }))
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
          $('#main table tbody').append($.template('tenants-form-invitee', { user: user }));
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
    if (!SHIELD.is('manager')) {
      $('#main').template('access-denied', { level: 'system', need: 'manager' });
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
        $('#main').html($($.template('tenants-form', { tenant: data, admin: true }))
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
              $('#main table tbody').append($.template('tenants-form-invitee', { user: user }));
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
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/auth/local/users',
      error: "Failed retrieving the list of local SHIELD users from the SHIELD API.",
      success: function (data) {
        $('#main').template('admin-users', { users: data });
      }
    });
    break; /* #!/admin/users */
    // }}}
  case "#!/admin/users/new": /* {{{ */
    if (!SHIELD.is('manager')) {
      $('#main').template('access-denied', { level: 'system', need: 'manager' });
      break;
    }
    $('#main').html($($.template('admin-users-new', {}))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();
        var $form = $(event.target);

        var payload = {
          name:     $form.find('[name=name]').val(),
          sysrole:  $form.find('[name=sysrole]').val(),
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
      }));
    break; // #!/admin/users/new
    // }}}
  case "#!/admin/users/edit": /* {{{ */
    if (!SHIELD.is('manager')) {
      $('#main').template('access-denied', { level: 'system', need: 'manager' });
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/auth/local/users/'+args.uuid,
      error: "Unable to retrieve user information from the SHIELD API.",
      success: function (data) {
        $('#main').html($($.template('admin-users-edit', { user: data }))
          .autofocus()
          .on('submit', 'form', function (event) {
            event.preventDefault();
            var $form = $(event.target);

            var payload = {
              name:    $form.find('[name=name]').val(),
              sysrole: $form.find('[name=sysrole]').val()
            };

            banner("Updating user...", "info");
            api({
              type: 'PATCH',
              url:  '/v2/auth/local/users/'+args.uuid,
              data: payload,
              success: function (data) {
                banner('User updated successfully.');
                goto("#!/admin/users");
              },
              error: function (xhr) {
                banner("Failed to update user", "error");
              }
            });
          }));
      }
    });
    break; // #!/admin/users/new
    // }}}

  case '#!/admin/stores': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('stores', {
      admin: true,
      stores: SHIELD._.global.stores
    });
    break; /* #!/admin/stores */
    // }}}
  case '#!/admin/stores/store': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    args.admin = true;
    $('#main').template('store', args);
    break; /* #!/admin/stores/store */
    // }}}
  case '#!/admin/stores/new': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    var data = { type: 'store' };
    $('#main').html($($.template('stores-form', { admin:  true }))
      .autofocus()
      .on('change', 'select[name="store.plugin"]', function (event) {
        data.plugin = $(event.target).val();
        console.log(data);
        $('#main .redraw.store').template('plugin-form-agent-selector', data);
      })
      .on('change', 'select[name="store.agent"]', function (event) {
        data.agent = $(event.target).val();
        console.log(data);
        $('#main .redraw.store').template('plugin-form-agent-selector', data);
      })
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        if (!$form.reset().validate().isOK()) { return; }

        var data = $form.serializeObject().store;
        data.threshold = readableToBytes(data.threshold);

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

    break; /* #!/admin/stores */
    // }}}
  case '#!/admin/stores/edit': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    var data = {
      admin: true,
      store: SHIELD.store(args.uuid, { includeGlobal: true })
    };
    $('#main').html($($.template('stores-form', data))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        if (!$form.reset().validate().isOK()) { return; }

        var data = $form.serializeObject().store;
        data.threshold = readableToBytes(data.threshold);
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

    break; /* #!/admin/stores/edit */
    // }}}
  case '#!/admin/stores/delete': /* {{{ */
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/global/stores/'+args.uuid,
      error: "Failed to retrieve storage system information from the SHIELD API.",
      success: function (store) {
        modal($($.template('stores-delete', { store: store }))
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

  case '#!/admin/sessions': /* {{{ */
    if (!SHIELD.is('admin')) {
      $('#main').template('access-denied', { level: 'system', need: 'admin' });
      break;
    }
    $('#main').template('loading');
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
      $('#main').template('sessions', { sessions: data, admin: true });
      }
    });
    break; /* #!/admin/sessions */
    // }}}
  case '#!/admin/sessions/delete': /* {{{ */
    if (!SHIELD.is('admin')) {
      $('#main').template('access-denied', { level: 'system', need: 'admin' });
      break;
    }
    api({
      type: 'GET',
      url:  '/v2/auth/sessions/'+args.uuid,
      error: "Failed to retrieve session information from the SHIELD API.",
      success: function (data) {
      modal($($.template('sessions-delete', { session: data }))
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
    if (!SHIELD.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').html($($.template('unlock', {}))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        $form.reset()
        var data = $form.serializeObject();
        if (data.master == "") {
          $form.error('unlock-master', 'missing');
          return;
        }

        api({
          type: 'POST',
          url:  '/v2/unlock',
          data: data,
          success: function (data) {
            goto("");
          },
          statusCode: {
            403: function () {
              $form.error('unlock-master', 'incorrect')
            },
            500: function (xhr) {
              $form.error(xhr.responseJSON);
            }
          },
          error: {}
        });
      }));
    break;
    // }}}

  default: /* 404 {{{ */
    $('#main').template('404', {
      wanted:  page,
      args:    argv,
      referer: referer,
    });
    return; /* 404 */
    // }}}
  }
  referer = page;
}

function redraw(complete) {
  if (complete && SHIELD.authenticated()) {
    $('#viewport').template('layout', {});
  }
  $('#hud').template('hud');
  $('.top-bar').template('top-bar', {
    user:    SHIELD._.user,
    tenants: SHIELD._.tenants,
    tenant:  SHIELD._.tenant
  });
  document.title = "SHIELD "+SHIELD.shield.env;
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
  new S.H.I.E.L.D.Database(function (db) {
    console.log('starting up...');
    viewSwitcher();

    /* ... watch the document hash for changes {{{ */
    $(window).on('hashchange', function (event) {
      dispatch(document.location.hash);
    }).trigger('hashchange');
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
      api({
        type: 'PATCH',
        url:  '/v2/auth/user/settings',
        data: { default_tenant: uuid }
      });
      SHIELD._.tenant = uuid;

      SHIELD.redraw();
      var page = document.location.hash.replace(/^(#!\/[^\/]*).*/, '$1');
      if (page == "#!/do")      { page = "#!/systems"; }
      if (page == "#!/tenants") { page = "#!/systems"; }
      if (page == "#!/admin")   { page = "#!/systems"; }
      goto(page);
    });
    $(document.body).on('click', '.top-bar .fly-out', function (event) {
      event.preventDefault();
      event.stopPropagation();
    });
    $(document.body).on('click', function (event) {
      $('.ephemeral').hide();
    });
    /* }}} */
    $(document.body).on('click', '.smudge span', function (event) {
      var $span = $(event.target).closest('span');
      var $fld  = $span.closest('.smudge').find('input');
      console.log($fld.attr('type'));
      switch ($fld.attr('type')) {
      case "text":
        $fld.attr('type', 'password');
        $span.text('show');
        break;

      case "password":
        $fld.attr('type', 'text');
        $span.text('hide');
        break;
      }
    });
    $(document.body).on('click', '.lean.selectable tbody tr', function (event) {
      var $tr = $(event.target).closest('tr');
      var $tbl = $tr.closest('.lean.selectable');

      if ($tr.hasClass('selected')) {
        $tbl.removeClass('selected');
        $tr.removeClass('selected');
      } else {
        $tbl.find('tr.selected').removeClass('selected');
        $tbl.addClass('selected');
        $tr.addClass('selected');
      }
    });

    /* global: show a task log in the next row down {{{ */
    $(document.body).on('click', 'a[href^="task:"]', function (event) {
      event.preventDefault();
      var uuid  = $(event.target).closest('a[href^="task:"]').attr('href').replace(/^task:/, '');
      var $ev   = $(event.target).closest('.event');
      var $task = $ev.find('.task');

      $task = $task.show()
                  .template('loading');

      api({
        type: 'GET',
        url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/tasks/'+uuid,
        error: "Failed to retrieve task information from the SHIELD API.",
        success: function (data) {
          $task.template('task', {
            task: data,
            restorable: data.type == "backup" && data.archive_uuid != "" && data.status == "done",
          });
          $(event.target).closest('li').hide();
        }
      });
    });
    /* }}} */
    /* global: close the expanded log, in a task log {{{ */
    $(document.body).on('click', '.task button[rel="close"]', function (event) {
      $ev = $(event.target).closest('.event');
      $ev.find('li.expand').show();
      $ev.find('.task').hide();
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
          url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/systems/'+uuid,
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
        url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/jobs/'+uuid+'/run',
        success: function () {
          banner('ad hoc backup job scheduled');
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
          url:  '/v2/tenants/'+SHIELD.activeTenant().uuid+'/archives/'+uuid+'/restore',
          success: function() {
            banner("restore operation started");
            redraw(false);
          },
          error: function () {
            banner("unable to schedule restore operation", "error");
          }
        });
      });
    });
    /* }}} */

    $(document.body).on('click', '.paginate .load-more', function (event) {
      console.log('loading more tasks...'); /* FIXME: need "loading" div... */
      event.preventDefault();

      $(event.target).closest('.paginate').find('.loading').show();

      var url    = $(event.target).closest('[data-url]').attr('data-url');
      var oldest = $(event.target).closest('[data-oldest]').attr('data-oldest');
      api({
        type: 'GET',
        url:  url.replace('{oldest}', oldest),
        error: 'Failed to retrieve tasks from the SHIELD API.',
        success: function (system) {
          var $outer = $(event.target).closest('.paginate').find('.results');
          for (var i = 0; i < system.tasks.length; i++) {
            //console.log('task: ', system.tasks[i]);
            //window.SHIELD.set('task', system.tasks[i]);
            $outer.append($.template('timeline-entry', system.tasks[i]));
            if (oldest > system.tasks[i].requested_at) {
                oldest = system.tasks[i].requested_at;
            }
          }
          $(event.target).closest('[data-oldest]').attr('data-oldest', oldest.toString());
          if (system.tasks.length == 0) {
            $(event.target).closest('.load-more').hide();
          }
          $(event.target).closest('.paginate').find('.loading').hide();
        }
      });
    });
  });
});
