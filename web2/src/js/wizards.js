;(function (exported, document, undefined) {

  var Wizard2 = function (e) {
    this.root     = $(e); /* root HTML element */
    this.handlers = {};   /* event handlers */

    /* set the number of steps as a class, for CSS progress bar styling */
    this.root.addClass('steps'+this.root.find('[data-step]').length.toString());
  };

  Wizard2.prototype.step = function (n, mode) {
    console.log('Wizard: transitioning to step %d (in "%s" mode)', n, mode);

    /* the before:$step hook fires before we show the step page */
    if ('before:'+n.toString() in this.handlers) {
      this.handler['before:'+n.toString()].call(this, {
        step: n,
        mode: mode
      });
    }

    var $progress = this.root.find('.progress li');
    this.root.find('[data-step]').each(function (i,e) {
      var $step = $(e);
      var num   = parseInt($step.attr('data-step'));
      var $li   = $($progress[num-1]);

      if (num < step) {
        $li.transitionClass('current', 'completed');
        $step.hide();

      } else if (num == step) {
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

    /* the after:$step hook fires before we show the step page */
    if ('after:'+n.toString() in this.handlers) {
      this.handler['after:'+n.toString()].call(this, {
        step: n,
        mode: mode
      });
    }

    $root.attr('data-on-step', step);
  };

  Wizard2.prototype.on = function (ev, fn) {
    this.handers[ev] = fn;
  };

  var Wizard = {
    setup: function ($root) {
      $root.addClass('steps'+$root.find('[data-step]').length.toString());
    },
    step: function ($root, step, mode) {
      console.log('Wizard: transitioning to step %d (in "%s" mode)', step, mode);

      var $progress = $root.find('.progress li');
      $root.find('[data-step]').each(function (i,e) {
        var $step = $(e);
        var num   = parseInt($step.attr('data-step'));
        var $li   = $($progress[num-1]);

        if (num < step) {
          $li.transitionClass('current', 'completed');
          $step.hide();
        } else if (num == step) {
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

      $root.attr('data-on-step', step);
    }
  };

  exported.DoConfigureWizard = function (root) {
    this.root = $(root);
    this._    = {};

    this.mount(this.root);
  };

  exported.DoConfigureWizard.prototype.step = function (next, mode) {
    Wizard.step(this.root, next, mode);
  };

  exported.DoConfigureWizard.prototype.mount = function ($root) {
    $root = $($root);
    Wizard.setup($root);
    Wizard.step($root, 1, 'choose');

    if ($root.data('do-configure-wizard:initialized')) {
      return; /* already mounted event handlers */
    }
    $root.data('do-configure-wizard:initialized', 'yes');

    console.log('mounting do-configure-wizard to ', $root);
    var self = this;
    $root
      .on('click', 'a[href^="step:"]', function (event) {
        event.preventDefault();
        var step = parseInt($(event.target).closest('[href]').attr('href').replace(/^step:/, ''));
        self.step(step);
      })
      .on('click', '[rel=prev]', function (event) {
        event.preventDefault();
        var now = parseInt($root.extract('on-step'));
        self.step(now - 1);
      })
      .on('click', '[rel=next]', function (event) {
        event.preventDefault();
        var now = parseInt($root.extract('on-step'));

        if (now + 1 == 5) { /* moving to Review step */
          self.root.find('.review').template('do-configure-review', self.val());
        }

        self.step(now + 1);
      })
      .on('click', '.choose.target [rel=new]', function (event) {
          event.preventDefault();
          self.step(2, 'create');
      })
      .on('click', '.choose.target tbody tr', function (event) {
        var $row = $(event.target).closest('tr');

        if (!$row.is('.selected')) { /* user wants to select ... */
          var uuid = $row.extract('target-uuid');
          console.log('choosing target: "%s"', uuid);
          self.step(2, uuid ? 'choose' : 'create');
        }
      })
      .on('click', '.choose.store [rel=new]', function (event) {
          event.preventDefault();
          self.step(3, 'create');
      })
      .on('click', '.choose.store tbody tr', function (event) {
        var $row = $(event.target).closest('tr');

        if (!$row.is('.selected')) { /* user wants to select ... */
          var uuid = $row.extract('store-uuid');
          console.log('choosing store: "%s"', uuid);
        }
      })

      .on('click', '.scheduling .optgroup [data-subform]', function (event) {
        var sub = $(event.target).extract('subform');
        $root.find('.scheduling .subform').hide();
        $root.find('.scheduling .subform#'+sub).show();
        $root.find('input[name=keep_days]')
               .attr('placeholder', $(event.target).extract('retain'));
      });
    ;
  };

  exported.DoConfigureWizard.prototype.val = function () {
    var val = {};

    /* TARGET */
    val.target = {};
    if (this.root.find('[data-step=2]').extract('mode') == 'choose') {
      val.target = { uuid: this.root.find('[data-step=2] tr.selected').extract('target-uuid') };

      var t = window.SHIELD.system(val.target.uuid);
      if (t) {
        val.target.name = t.name;
        val.target.plugin = t.plugin;
      }

    } else {
      val.target = this.root.find('[data-step=2] form').serializeObject().target;
    }

    /* STORE */
    val.store = {};
    if (this.root.find('[data-step=3]').extract('mode') == 'choose') {
      val.store = { uuid: this.root.find('[data-step=3] tr.selected').extract('store-uuid') };

      var t = window.SHIELD.store(val.store.uuid);
      if (t) {
        val.store.name = t.name;
        val.store.plugin = t.plugin;
      }

    } else {
      val.store = this.root.find('[data-step=3] form').serializeObject().store;
    }

    /* SCHEDULING */
    val.schedule = {
      spec: this.root.find('[data-step=4] form').timespec(),
      keep: this.root.find('[name=keep_days]').val()
    };
    if (!val.schedule.keep) {
      val.schedule.keep = this.root.find('[data-step=4] [name=keep_days]').attr('placeholder');
    }

    return val;
  };

  exported.DoAdHocWizard = function (root) {
    this.root = $(root);
    this._    = {};

    this.mount(this.root);
  };

  exported.DoAdHocWizard.prototype.step = function (next, mode) {
    Wizard.step(this.root, next, mode);
  };

  exported.DoAdHocWizard.prototype.mount = function ($root) {
    $root = $($root);
    Wizard.setup($root);
    Wizard.step($root, 1);

    if ($root.data('do-ad-hoc-wizard:initialized')) {
      return; /* already mounted event handlers */
    }
    $root.data('do-ad-hoc-wizard:initialized', 'yes');

    console.log('mounting do-ad-hoc-wizard to ', $root);
    var self = this;
    $root
      .on('click', 'a[href^="step:"]', function (event) {
        event.preventDefault();
        var step = parseInt($(event.target).closest('[href]').attr('href').replace(/^step:/, ''));
        self.step(step);
      })
      .on('click', '[rel=prev]', function (event) {
        event.preventDefault();
        var now = parseInt($root.extract('on-step'));
        self.step(now - 1);
      })
      .on('click', '[rel=next]', function (event) {
        event.preventDefault();
        var now = parseInt($root.extract('on-step'));
        self.step(now + 1);
      })
      .on('click', '.choose.target tbody tr', function (event) {
        var $row = $(event.target).closest('tr')

        if (!$row.is('.selected')) { /* user wants to select ... */
          var uuid = $row.extract('target-uuid');
          console.log('choosing target: "%s"', uuid);
        }
      })
      .on('click', '.choose.store tbody tr', function (event) {
        /* ... */
      })
    ;
  };

  exported.DoRestoreWizard = function (root) {
    this.root = $(root);
    this._    = {};

    this.mount(this.root);
  };

  exported.DoRestoreWizard.prototype.step = function (next, mode) {
    Wizard.step(this.root, next, mode);
  };

  exported.DoRestoreWizard.prototype.mount = function ($root) {
    $root = $($root);
    Wizard.setup($root);
    Wizard.step($root, 1);

    if ($root.data('do-restore-wizard:initialized')) {
      return; /* already mounted event handlers */
    }
    $root.data('do-restore-wizard:initialized', 'yes');

    console.log('mounting do-restore-wizard to ', $root);
    var self = this;
    $root
      .on('click', 'a[href^="step:"]', function (event) {
        event.preventDefault();
        var step = parseInt($(event.target).closest('[href]').attr('href').replace(/^step:/, ''));
        self.step(step);
      })
      .on('click', '[rel=prev]', function (event) {
        event.preventDefault();
        var now = parseInt($root.extract('on-step'));
        self.step(now - 1);
      })
      .on('click', '[rel=next]', function (event) {
        event.preventDefault();
        var now = parseInt($root.extract('on-step'));
        self.step(now + 1, 'choose');
      })
      .on('click', '.choose.target tbody tr', function (event) {
        var $row = $(event.target).closest('tr');

        if (!$row.is('.selected')) { /* user wants to select ... */
          var uuid = $row.extract('target-uuid');
          console.log('choosing target: "%s"', uuid);
        }
      })
      .on('click', '.choose.archive tbody tr', function (event) {
        /* ... */
      })
    ;
  };

})(window, document);
