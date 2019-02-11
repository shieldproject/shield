;(function ($, document, window, undefined) {

  $(function () {

    $(document.body)

      /* Hide the fly-out menu if we click _anywhere_ else */
      .on('click', function (event) { /* {{{ */
        $('.ephemeral').hide();
      }) /* }}} */

      /* Smudge fields */
      .on('click', 'form .smudge span', function (event) { /* {{{ */
        /*
           A smudge field displays initially as a password field, but features
           an in-control widget for toggling the visibility of the entered
           secret.  This allows users to verify that they typed in a key,
           without resorting to the double-confirmation method.

         */
        var $span = $(event.target).closest('span');
        var $fld  = $span.closest('.smudge').find('input');

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
      })/* }}} */

      /* Lean Table vs. Card View switcher */
      .on('click', '.switch-me .switcher a[href^="switch:"]', function (event) { /* {{{ */
        event.preventDefault();
        var view  = $(event.target).closest('a[href^="switch:"]').attr('href').replace(/^switch:/, '');
        var swtch = $(event.target).closest('.switch-me');

        $.each(swtch[0].className.split(/\s+/), function (i, cls) {
          if (cls.match(/-view$/)) {
            swtch.removeClass(cls);
          }
        });
        localStorage.setItem('view-preference', view);
        swtch.addClass(view);
      }) /* }}} */

      /* Account Fly-Out Menu */
      .on('click', '.top-bar a[rel=account]', function (event) { /* {{{ */
        event.preventDefault();
        event.stopPropagation();
        $('.top-bar .flyout').toggle();
      }) /* }}} */
      .on('click', '.top-bar .fly-out', function (event) { /* {{{ */
        /* don't propagate to the top-level click handler
           since that will hide the menu we just clicked on. */
        event.preventDefault();
        event.stopPropagation();
      }) /* }}} */
      .on('click', '.top-bar a[href^="switchto:"]', function (event) { /* {{{ */
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
      }) /* }}} */

      /* Selectable Table UI Widget */
      .on('click', '.lean.selectable tbody tr', function (event) { /* {{{ */
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
      }) /* }}} */

      /* "Run Job" links (href="run:...") */
      .on('click', 'a[href^="run:"], button[rel^="run:"]', function (event) { /* {{{ */
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
      }) /* }}} */

      /* Task Pagination */
        .on('click', '.paginate .load-more', function (event) { /* {{{ */
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
        }) /* }}} */

      /* Tasks View */
      .on('click', 'a[href^="task:"]', function (event) { /* {{{ */
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
      }) /* }}} */
      .on('click', '.task button[rel="close"]', function (event) { /* {{{ */
        $ev = $(event.target).closest('.event');
        $ev.find('li.expand').show();
        $ev.find('.task').hide();
      }) /* }}} */
      .on('click', '.task button[rel^="annotate:"]', function (event) { /* {{{ */
        $(event.target).closest('.task').find('form.annotate').toggle();
      }) /* }}} */
      .on('submit', '.task form.annotate', function (event) { /* {{{ */
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
      }) /* }}} */

      /* "Restore Archive" links (rel="restore:...") */
      .on('click', '.task button[rel^="restore:"]', function (event) { /* {{{ */
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
      }) /* }}} */
    ;
  });

})(jQuery, document, window);
