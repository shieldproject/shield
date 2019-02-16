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
          self.root.find('.review').template('do-configure-review', self.data());
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
        self.root.find('input[name="job.keep_days"]')
            .attr('placeholder', $(event.target).extract('retain'));
      })

      .on('click', prefix+' button.final', function (event) {
        event.preventDefault();
        $(event.target).addClass('submitting');

        api({
          type: 'POST',
          url:  '/v2/tenants/'+AEGIS.current.uuid+'/systems',
          data: self.data(),

          error: "Failed to create system via the SHIELD API.",
          complete: function () {
            $(event.target).removeClass('submitting');
          },
          success: function (data) {
            goto('#!/systems/system:uuid:'+data.uuid);
          }
        });
      })
    ;
  };

  doConfigureWizard.prototype.data = function () {
    var data = {};
    var step = parseInt(this.root.attr('data-on-step'));

    /* TARGET */
    if (step >= 2) {
      data.target = {};
      if (this.root.find('[data-step=2]').extract('mode') == 'choose') {
        data.target.uuid = this.root.find('[data-step=2] tr.selected').extract('target-uuid');

        var t = AEGIS.system(data.target.uuid);
        if (t) {
          data.target.name   = t.name;
          data.target.plugin = t.plugin;
        }

      } else {
        data.target = this.root.find('[data-step=2] form').serializeObject().target;
      }
    }

    /* STORE */
    if (step >= 3) {
      data.store = {};
      if (this.root.find('[data-step=3]').extract('mode') == 'choose') {
        data.store.uuid = this.root.find('[data-step=3] tr.selected').extract('store-uuid');

        var t = AEGIS.store(data.store.uuid);
        if (t) {
          data.store.name = t.name;
          data.store.plugin = t.plugin;
        }

      } else {
        data.store = this.root.find('[data-step=3] form').serializeObject().store;
      }
    }

    /* SCHEDULING */
    if (step >= 4) {
      data.job = this.root.find('[data-step=4] form').serializeObject().job;
      data.job.schedule = this.root.find('[data-step=4] form').timespec();
      if (!data.job.keep_days) {
        data.job.keep_days = this.root.find('[data-step=4] [name="job.keep_days"]').attr('placeholder');
      }
      data.job.keep_days = parseInt(data.job.keep_days);
      data.job.fixed_key = !data.job.randomize_keys;
      delete data.job.randomize_keys;
    }

    return data;
  };

  exported.DoConfigureWizard = doConfigureWizard;

  var doAdHocWizard = function ($main, prefix) {
    this.root = $main.find(prefix);
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
          var data = self.data();
          if (!data.target) { return; }
          $('#main .do-backup .redraw.jobs').template('do-backup-choose-job', data);
        }

        if (now + 1 == 4) { /* moving to Review step */
          var data = self.data();
          if (!data.target || !data.job) { return; }
          $('#main .do-backup .review').template('do-backup-review', data);
        }

        step(self.root, now + 1);
      })
      .on('click', prefix+' button.final', function (event) {
        event.preventDefault();
        $(event.target).addClass('submitting');

        var data = self.data();
        api({
          type: 'POST',
          url:  '/v2/tenants/'+AEGIS.current.uuid+'/jobs/'+data.job.uuid+'/run',
          data: {},

          error: "Failed to create system via the SHIELD API.",
          complete: function () {
            $(event.target).removeClass('submitting');
          },
          success: function () {
            goto('#!/systems/system:uuid:'+data.target.uuid);
          }
        });
      })
    ;
  };

  doAdHocWizard.prototype.data = function () {
    var data = {};
    var step = parseInt(this.root.attr('data-on-step'));

    /* TARGET */
    if (step >= 2) {
      data.target = {};
      data.target.uuid = this.root.find('[data-step=2] tr.selected').extract('target-uuid');
    }

    /* JOB */
    if (step >= 3) {
      data.job = {};
      data.job.uuid = this.root.find('[data-step=3] tr.selected').extract('job-uuid');
    }

    return data;
  };

  exported.DoAdHocWizard = doAdHocWizard;

  var doRestoreWizard = function ($main, prefix) {
    this.root = $main.find(prefix);
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
          var data = self.data();
          if (!data.target) { return; }
          $('#main .do-restore .redraw.archives').template('do-restore-choose-archive', data);
        }

        if (now + 1 == 4) { /* moving to Review step */
          var data = self.data();
          if (!data.target || !data.archive) { return; }
          $('#main .do-restore .review').template('do-restore-review', data);
        }

        step(self.root, now + 1);
      })
      .on('click', prefix+' button.final', function (event) {
        event.preventDefault();
        $(event.target).addClass('submitting');

        var data = self.data();
        api({
          type: 'POST',
          url:  '/v2/tenants/'+AEGIS.current.uuid+'/archives/'+data.archive.uuid+'/restore',
          data: {},

          error: "Failed to create system via the SHIELD API.",
          complete: function () {
            $(event.target).removeClass('submitting');
          },
          success: function () {
            goto('#!/systems/system:uuid:'+data.target.uuid);
          }
        });
      })
    ;
  };

  doRestoreWizard.prototype.data = function () {
    var data = {};
    var step = parseInt(this.root.attr('data-on-step'));

    /* TARGET */
    if (step >= 2) {
      data.target = {};
      data.target.uuid = this.root.find('[data-step=2] tr.selected').extract('target-uuid');
    }

    /* ARCHIVE */
    if (step >= 3) {
      data.archive = {};
      data.archive.uuid = this.root.find('[data-step=3] tr.selected').extract('archive-uuid');
    }

    return data;
  };

  exported.DoRestoreWizard = doRestoreWizard;

})(window, window.jQuery, document);
