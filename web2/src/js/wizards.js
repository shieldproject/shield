;(function (exported, jQuery, document, undefined) {

  var step = function (root, n, mode) {
    console.log('step()', { root: root, n: n, mode: mode });
    console.log('Wizard: transitioning to step %d (in "%s" mode)', n, mode);

    var $progress = root.find('.progress li');
    root.find('[data-step]').each(function (i,e) {
      var $step = $(e);
      var num   = parseInt($step.attr('data-step'));
      var $li   = $($progress[num-1]);

      if (num < n) {
        $li.transitionClass('current', 'completed');
        $step.hide();

      } else if (num == n) {
        $li.transitionClass('completed', 'current');
        if (mode) {
          $step.attr('data-mode', mode);
        }
        $step.autofocus().show();

      } else {
        $li.removeClass('current completed');
        $step.hide();
      }
    });

    root.attr('data-on-step', n);
  };

  var doConfigureWizard = function ($main, prefix) {
    this.root = $main.find(prefix);
    this.root.addClass('steps'+this.root.find('[data-step]').length.toString());
    step(this.root, 1, 'choose');

    if ($main.data('do-configure-wizard:initialized')) {
      return; /* already mounted event handlers */
    }
    $main.data('do-configure-wizard:initialized', 'yes');

    var self = this;
    $main
      .on('click', prefix+' a[href^="step:"]', function (event) {
        event.preventDefault();
        step(self.root, parseInt($(event.target).closest('[href]').attr('href').replace(/^step:/, '')));
      })
      .on('click', prefix+' [rel=prev]', function (event) {
        event.preventDefault();
        var now = parseInt(self.root.extract('on-step'));
        step(self.root, now - 1);
      })
      .on('click', prefix+' [rel=next]', function (event) {
        event.preventDefault();
        var now = parseInt(self.root.extract('on-step'));

        if (now + 1 == 3 && self.root.find('[data-step=2]').extract('mode') == 'create') {
          var $form = self.root.find('[data-step=2] form');
          if (!$form.reset().validate().isOK()) { return; }
        }

        if (now + 1 == 4 && self.root.find('[data-step=3]').extract('mode') == 'create') {
          var $form = self.root.find('[data-step=3] form');
          if (!$form.reset().validate().isOK()) { return; }
        }

        if (now + 1 == 5) { /* moving to Review step */
          var val = {};

          /* TARGET */
          val.target = {};
          if (self.root.find('[data-step=2]').extract('mode') == 'choose') {
            val.target = { uuid: self.root.find('[data-step=2] tr.selected').extract('target-uuid') };

            var t = window.SHIELD.system(val.target.uuid);
            if (t) {
              val.target.name = t.name;
              val.target.plugin = t.plugin;
            }

          } else {
            val.target = self.root.find('[data-step=2] form').serializeObject().target;
          }

          /* STORE */
          val.store = {};
          if (self.root.find('[data-step=3]').extract('mode') == 'choose') {
            val.store = { uuid: self.root.find('[data-step=3] tr.selected').extract('store-uuid') };

            var t = window.SHIELD.store(val.store.uuid);
            if (t) {
              val.store.name = t.name;
              val.store.plugin = t.plugin;
            }

          } else {
            val.store = self.root.find('[data-step=3] form').serializeObject().store;
          }

          /* SCHEDULING */
          val.schedule = {
            spec: self.root.find('[data-step=4] form').timespec(),
            keep: self.root.find('[name=keep_days]').val()
          };
          if (!val.schedule.keep) {
            val.schedule.keep = self.root.find('[data-step=4] [name=keep_days]').attr('placeholder');
          }
          self.root.find('.review').template('do-configure-review', val);
        }

        step(self.root, now + 1);
      })
      .on('click', prefix+' .choose.target [rel=new]', function (event) {
          event.preventDefault();
          step(self.root, 2, 'create');
      })
      .on('click', prefix+' .choose.target tbody tr', function (event) {
        var $row = $(event.target).closest('tr');

        if (!$row.is('.selected')) { /* user wants to select ... */
          var uuid = $row.extract('target-uuid');
          console.log('choosing target: "%s"', uuid);
          step(self.root, 2, uuid ? 'choose' : 'create');
        }
      })
      .on('click', prefix+' .choose.store [rel=new]', function (event) {
          event.preventDefault();
          step(self.root, 3, 'create');
      })
      .on('click', prefix+' .choose.store tbody tr[data-store-uuid]', function (event) {
        var $row = $(event.target).closest('tr');

        if (!$row.is('.selected')) { /* user wants to select ... */
          var uuid = $row.extract('store-uuid');
          console.log('choosing store: "%s"', uuid);
          step(self.root, 3, uuid ? 'choose' : 'create');
        }
      })

      .on('click', prefix+' .scheduling .optgroup [data-subform]', function (event) {
        var sub = $(event.target).extract('subform');
        self.root.find('.scheduling .subform').hide();
        self.root.find('.scheduling .subform#'+sub).show();
        self.root.find('input[name=keep_days]')
            .attr('placeholder', $(event.target).extract('retain'));
      })
    ;

    return this;
  };

  exported.doConfigureWizard = doConfigureWizard;

  var doAdHocWizard = function (root) {
    this.root = $(root);
  };

  doAdHocWizard.prototype.mount = function (e, prefix) {
    var $main = $(e);
    this.root.addClass('steps'+this.root.find('[data-step]').length.toString());
    step(this.root, 1);

    if ($main.data('do-ad-hoc-wizard:initialized')) {
      return; /* already mounted event handlers */
    }
    $main.data('do-ad-hoc-wizard:initialized', 'yes');

    var self = this;
    $main
      .on('click', prefix+' a[href^="step:"]', function (event) {
        event.preventDefault();
        step(self.root, parseInt($(event.target).closest('[href]').attr('href').replace(/^step:/, '')));
      })
      .on('click', prefix+' [rel=prev]', function (event) {
        event.preventDefault();
        var now = parseInt(self.root.extract('on-step'));
        step(self.root, now - 1);
      })
      .on('click', prefix+' [rel=next]', function (event) {
        event.preventDefault();
        var now = parseInt(self.root.extract('on-step'));

        if (now + 1 == 3) {
          var uuid = self.root.find('[data-step=2] tr.selected').extract('target-uuid');
          console.log('selecting target %s', uuid);
          if (!uuid) { return; }

          $('#main .do-backup .redraw.jobs').template('do-backup-choose-job', {
            selected_target: uuid,
          });
        }

        if (now + 1 == 4) { /* moving to Review step */
          var data = {};

          data.selected_target = self.root.find('[data-step=2] tr.selected').extract('target-uuid');
          if (!data.selected_target) { return; }
          data.selected_job = self.root.find('[data-step=3] tr.selected').extract('job-uuid');
          if (!data.selected_job) { return; }

          $('#main .do-backup .review').template('do-backup-review', data);
        }

        step(self.root, now + 1);
      })
    ;

    return self;
  };

  exported.DoAdHocWizard = doAdHocWizard;

  var doRestoreWizard = function (root) {
    this.root = $(root);
  };

  doRestoreWizard.prototype.mount = function (e, prefix) {
    var $main = $(e);
    this.root.addClass('steps'+this.root.find('[data-step]').length.toString());
    step(this.root, 1);

    if ($main.data('do-restore-wizard:initialized')) {
      return; /* already mounted event handlers */
    }
    $main.data('do-restore-wizard:initialized', 'yes');

    var self = this;
    $main
      .on('click', prefix+' a[href^="step:"]', function (event) {
        event.preventDefault();
        step(self.root, parseInt($(event.target).closest('[href]').attr('href').replace(/^step:/, '')));
      })
      .on('click', prefix+' [rel=prev]', function (event) {
        event.preventDefault();
        var now = parseInt(self.root.extract('on-step'));
        step(self.root, now - 1);
      })
      .on('click', prefix+' [rel=next]', function (event) {
        event.preventDefault();
        var now = parseInt(self.root.extract('on-step'));

        if (now + 1 == 3) {
          var uuid = self.root.find('[data-step=2] tr.selected').extract('target-uuid');
          console.log('selecting target %s', uuid);
          if (!uuid) { return; }

          $('#main .do-restore .redraw.archives').template('do-restore-choose-archive', {
            selected_target: uuid,
          });
        }

        if (now + 1 == 4) { /* moving to Review step */
          var data = {};

          data.selected_target = self.root.find('[data-step=2] tr.selected').extract('target-uuid');
          if (!data.selected_target) { return; }

          data.selected_archive = self.root.find('[data-step=3] tr.selected').extract('archive-uuid');
          if (!data.selected_archive) { return; }

          $('#main .do-restore .review').template('do-restore-review', data);
        }

        step(self.root, now + 1);
      })
    ;
  };

  exported.DoRestoreWizard = doRestoreWizard;

})(window, window.jQuery, document);
