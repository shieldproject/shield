// vim:et:sts=2:ts=2:sw=2
var referer;

function divert(page) { // {{{
  if (page.match(/^#!\/(login|logout|cliauth)$/)) {
    /* never divert these pages */
    return page;
  }

  if (!AEGIS.authenticated()) {
    console.log('session not authenticated; diverting to #!/login page...');
    return "#!/login";
  }

  if (!page || page == "") {
    return AEGIS.is('engineer') ? '#!/admin' : '#!/systems';
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

  console.log('dispatching to %s (from "%s")...', page, dest);

  var top = page.replace(/^(#!\/[^\/]+)(\/.*)?$/, '$1');
  $('nav li.current').removeClass('current');
  $('nav a[href="'+top+'"]').closest('li').addClass('current');

  Scratch.clear();
  Scratch.track('redrawable', true);
  switch (page) {

  case "#!/login": /* {{{ */
    (function () {
      var progress = function (how) {
        $('#viewport').find('#logging-in').remove();
        $('#viewport').append($.template('logging-in', {auth: how}));
      };

      $.when(
        $.ajax({ type: 'GET', url: '/v2/auth/providers?for=web' }),
        $.ajax({ type: 'GET', url: '/v2/info' })
      ).then(function (providers, info) {
        $('#viewport').html($($.template('login', { providers: providers[0],
                                                    info:      info[0] }))
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
      }, function () {
        console.dir(arguments);
        $('#viewport').template('BOOM');
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

  case "#!/do/backup": /* {{{ */
    Scratch.track('redrawable', false);
    $('#main').template('do-backup');
    $('#main .do-backup').trigger('wizard:step', { to: 1 });
    break; /* #!/do/backup */
    // }}}
  case "#!/do/restore": /* {{{ */
    Scratch.track('redrawable', false);
    $('#main').template('do-restore');
    $('#main .do-restore').trigger('wizard:step', { to: 1 });
    break; /* #!/do/restore */
    // }}}
  case "#!/do/configure": /* {{{ */
    Scratch.track('redrawable', false);
    if (!AEGIS.is('engineer')) {
      $('#main').template('access-denied', { need: 'engineer' });
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
      $('#main .do-configure').trigger('wizard:step', { to: 1 });
      $('#main .optgroup').optgroup();
      $('#main .scheduling [data-subform=schedule-daily]').trigger('click');
    }, 150);
    break; /* #!/do/configure */
    // }}}

  case "#!/systems": /* {{{ */
    $('#main').template('systems');
    break; /* #!/systems */
    // }}}
  case '#!/systems/system': /* {{{ */
    $('#main').template('system', args);
    $('#main .paginate .load-more').trigger('click');
    api({
      type:     'GET',
      url:      '/v2/systems/'+args.uuid+'/config'
    }, true).then(function (config) {
      args.config = config;
      $('#main').template('system', args);
    });
    break; /* #!/systems/system */
    // }}}
  case '#!/systems/edit': /* {{{ */
    Scratch.track('redrawable', false);
    if (!AEGIS.is('engineer')) {
      $('#main').template('access-denied', { need: 'engineer' });
      break;
    }
    $('#main').html($($.template('targets-form', args))
      .autofocus()
      .on('submit', 'form', function (event) {
        event.preventDefault();

        var $form = $(event.target);
        if (!$form.reset().validate().isOK()) { return; }

        var data = $form.serializeObject();
        $form.submitting(true);
        api({
          type: 'PUT',
          url:  '/v2/targets/'+args.uuid,
          data: data,
          success: function () {
            goto("#!/systems/system:uuid:"+args.uuid);
          },
          error: function (xhr) {
            $form.submitting(false);
            $form.error(xhr.responseJSON);
          }
        });
      }));

    break; /* #!/systems/edit */
    // }}}

  case '#!/buckets': /* {{{ */
    $('#main').template('buckets');
    break; /* #!/buckets */
    // }}}

  case '#!/admin': /* {{{ */
    if (!AEGIS.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('admin');
    break; /* #!/admin */
    // }}}
  case '#!/admin/agents': /* {{{ */
    if (!AEGIS.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    Scratch.track('redrawable', false);
    $('#main').template('loading');
    api({
      type: 'GET',
      url:  '/v2/agents',
      error: "Failed retrieving the list of agents from the SHIELD API.",
      success: function (data) {
        $('#main').html($($.template('agents', data))
          .on('click', 'a[rel]', function (event) {
            event.preventDefault();

            var action = $(event.target).closest('a[rel]').attr('rel');
            if (action == 'hide' || action == 'show') {
              api({
                type: 'POST',
                url:  '/v2/agents/'+$(event.target).extract('agent-uuid')+'/'+action,
                error: "Unable to "+action+" agent via the SHIELD API.",
                success: function () { reload(); }
              });

            } else if (action == 'resync') {
              api({
                type: 'POST',
                url:  '/v2/agents/'+$(event.target).extract('agent-uuid')+'/resync',
                error: "Resynchronization request failed",
                success: function () {
                  banner("Resynchronization of agent underway");
                }
              });
            } else if (action == 'delete') {
              var agent_uuid = $(event.target).extract('agent-uuid');
              modal($($.template('agents-delete', { agent: data.agents[0] }))
              .on('click', '[rel="yes"]', function (event) {
                event.preventDefault();
                api({
                  type: 'DELETE',
                  url:  '/v2/agents/'+agent_uuid,
                  error: "Unable to delete agent",
                  complete: function () {
                    modal(true);
                  },
                  success: function (event) {
                    goto('#!/admin/agents');
                  }
                });
              })
              .on('click', '[rel="close"]', function (event) {
                  modal(true);
                  goto('#!/admin/agents');
                })
              );
            }
          }));
      }
    });
    break; /* #!/admin/agents */
    // }}}
  case '#!/admin/auth': /* {{{ */
    if (!AEGIS.is('engineer')) {
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
    if (!AEGIS.is('engineer')) {
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

  case '#!/admin/users': /* {{{ */
    if (!AEGIS.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    $('#main').template('loading');
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
    Scratch.track('redrawable', false);
    if (!AEGIS.is('manager')) {
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
        $form.submitting(true);
        api({
          type: 'POST',
          url:  '/v2/auth/local/users',
          data: payload,
          success: function (data) {
            banner('New user created successfully.');
            goto("#!/admin/users");
          },
          error: function (xhr) {
            $form.submitting(false);
            banner("Failed to create new user", "error");
          }
        });
      }));
    break; // #!/admin/users/new
    // }}}
  case "#!/admin/users/edit": /* {{{ */
    Scratch.track('redrawable', false);
    if (!AEGIS.is('manager')) {
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
            var data = $form.serializeObject();

            $form.reset();
            if (data.password != data.confirm) {
              $form.error('confirm', 'mismatch');
            }

            if (!$form.isOK()) {
              return;
            }
            delete data.confirm;

            if ($form.find('[name=password]').val()==""){
              var payload = {
                name:    $form.find('[name=name]').val(),
                sysrole: $form.find('[name=sysrole]').val(),
              };
            } else {
              var payload = {
                name:    $form.find('[name=name]').val(),
                sysrole: $form.find('[name=sysrole]').val(),
                password: $form.find('[name=password]').val()
              };
            }

            banner("Updating user...", "info");
            $form.submitting(true);
            api({
              type: 'PATCH',
              url:  '/v2/auth/local/users/'+args.uuid,
              data: payload,
              success: function (data) {
                banner('User updated successfully.');
                goto("#!/admin/users");
              },
              error: function (xhr) {
                $form.submitting(false);
                banner("Failed to update user", "error");
              }
            });
          }));
      }
    });
    break; // #!/admin/users/new
    // }}}

  case '#!/admin/sessions': /* {{{ */
    if (!AEGIS.is('admin')) {
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
    if (!AEGIS.is('admin')) {
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

  case '#!/admin/fixups': /* {{{ */
    if (!AEGIS.is('engineer')) {
      $('#main').template('access-denied', { level: 'system', need: 'engineer' });
      break;
    }
    Scratch.track('redrawable', false);
    $('#main').template('loading');
    api({
      type: 'GET',
      url:  '/v2/fixups',
      error: "Failed retrieving the list of data fixups from the SHIELD API.",
      success: function (data) {
        $('#main').template('fixups', { fixups: data })
          .on('click', 'a[href^="apply-fixup:"]', function (event) {
            event.preventDefault();

            var a = $(event.target).closest('a[href^="apply-fixup:"]');
            var id = a.attr('href').replace(/^apply-fixup:/, '')
            api({
              type: 'POST',
              url:  '/v2/fixups/'+id+'/apply',
              error: 'Unable to apply data fixup "'+id+'".',
              success: function () {
                banner('Fixup "'+id+'" applied successfully!');
                reload();
              }
            });
          });
      }
    });
    break; /* #!/admin/fixups */
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

function goto(page) {
  if (document.location.hash == page) {
    $(document.body).trigger('shield:navigate', page);
  } else {
    document.location.hash = page;
  }
}
function reload() {
  goto(document.location.hash);
}

$(function () {
  $('#viewport').template('loading');
  window.AEGIS = $.aegis();
  window.AEGIS.subscribe()
    .then(
      function () {
        document.title = "SHIELD "+AEGIS.shield.env;
        $('.top-bar').template('top-bar');
        if (AEGIS.authenticated()) {
          $('#viewport').template('layout');
          $('#side-bar').template('side-bar');
          $('#hud').template('hud');
        }
        $(document.body)
          .on('shield:navigate', function (event, to) {
            api(); // clear outstanding API calls
            dispatch(to);
          })
          .on('click', 'a[href^="#"]:not([rel])', function (event) {
            goto($(event.target).closest('[href]').attr('href'));
          });
        $(window).on('hashchange', function (event) {
          $(document.body).trigger('shield:navigate', document.location.hash);
        }).trigger('hashchange');
      },
      function () {
        console.log('AEGIS subscription setup failed, redirecting to login page...');
        dispatch("#!/login");
      });
});
