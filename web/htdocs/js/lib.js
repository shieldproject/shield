;(function ($, exported, document, undefined) {

  exported.h = (function () {
    var e = document.createElement('div');
    return function (s) {
      e.innerText = (typeof(s) === 'undefined' ? '' : s).toString();
      return e.innerHTML.replace(/&lt;(\/?redacted)&gt;/g, '<$1>');
    };
  })();

  exported.md = (function () {
    var converter = new showdown.Converter({
      omitExtraWLInCodeBlocks:   true,
      simplifiedAutoLink:        true,
      literalMidWordUnderscores: true,
      strikethrough:             true,
      tables:                    true,
      simpleLineBreaks:          true,
      openLinksInNewWindow:      true
    });
    return function (s) {
      /* h() translates newlines into <br> tags, so we
         have to translate them back to avoid confusing
         the markdown parser / reformatter. */
      return converter.makeHtml(h(s).replace(/<br>/g, "\n"));
    };
  })();

  $.all = function (l, fn) {
    var ok = true;
    $.each(l, function (i, o) {
      if (!fn(o)) { ok = false; }
    })
    return ok
  };

  /***************************************************
    pluralize(n, word [, words]) - Pluralize a number + unit.

    In the two-argument form, `word` will have an 's' appended to it if
    `n` is any quantity other than 1.  In the three-argument form, `word`
    is used as the singular unit, and `words` as the plural.

   ***************************************************/
  exported.pluralize = function (n, word, words) { // {{{
    if (typeof(n) === 'undefined') {
      n = 0;
    }
    if (n == 1) {
      return '1 '+word;
    }
    if (typeof(words) === 'undefined') {
      return n.toString()+' '+word+'s';
    }
    return n.toString()+' '+words;
  }
  // }}}

  exported.tnow = function () {
    var d = new Date();
    return d.getTime() / 1000;
  };

  /***************************************************
    tparse(d) - Parse `d` intelligently.

    If `d` is already a Date, it gets returned as-is. Strings are parsed
    according to the goofy way the SHIELD API formats dates/times.

   ***************************************************/
  exported.tparse = function (d) { // {{{
    if (typeof d == "string") {
      // because we insist on doing bad things with date formatting in the API...
      m = /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/.exec(d);
      if (!m) {
        console.log("'%s' doesn't look like a date/time string...", d);
        return ""
      }

      d = new Date();
      d.setTime(Date.UTC(parseInt(m[1]),   // year
                        parseInt(m[2])-1, // month
                        parseInt(m[3]),   // day
                        parseInt(m[4]),   // hour
                        parseInt(m[5]),   // minute
                        parseInt(m[6]))); // second
      return d;
    }

    if (!(d instanceof Date)) {
      var _d = new Date()
      if (!isNaN(d)) {
        _d.setTime(d * 1000);
      }
      return _d;
    }

    return d;
  }
  // }}}


  /***************************************************
    tdiff(t1, t2) - Determine the number of seconds between two times

    The `t1` and `t2` arguments will first be passed through tparse(), so
    you can safely pass strings and Date objects with reckless abandon.

   ***************************************************/
  exported.tdiff = function (t1, t2) { // {{{
    return parseInt((exported.tparse(t2).getTime() / 1000)
                  - (exported.tparse(t1).getTime() / 1000));
  }
  // }}}


  /***************************************************
    duration(s) - Generate a human-readable duration, given a number of seconds.

   ***************************************************/
  exported.duration = function (t) { // {{{
    var d, h, m;
      t = parseInt(t);
      if (t < 0) { t = 0; }
      if (t < 60) {
        return t.toString() + "s";
      }
      if (t < 60 * 60) {
        return parseInt(t / 60).toString() + "m";
      }
      if (t < 60 * 60 * 24) {
        h = parseInt(t / (60 * 60));
        m = (t - h*(60 * 60)) / 60;
        return parseInt(h).toString() + "h " + parseInt(m).toString() + "m";
      }
      if (t < 60 * 60 * 24 * 7) {
        d = parseInt(t / (60 * 60 * 24));
        h = (t - d*(60 * 60 * 24)) / (60 * 60);
        return parseInt(d).toString() + "d " + parseInt(h).toString() + "h";
      }
      return parseInt(t / (60 * 60 * 24)).toString() + "d";
  }
  // }}}


  /***************************************************
    trelative(d, threshold, fmt) - Intelligent "from now" durations

    If the time between `d` and now is larger than `threshold`, an absolute
    date/time string, of the format `fmt` (as interpreted by `strftime()`)
    will be returned.

    Otherwise, a duration of the difference between `d` and now is returned,
    suffixed with the text " ago".

    This is useful for providing accuracy at both ends of a relative date/time
    spectrum - small differences will render as '5m ago' or '12h ago', whereas
    large differences just show up as the full date/time.

   ***************************************************/
  exported.trelative = function (d, threshold, fmt) { // {{{
    d = exported.tparse(d);
    var now = new Date();
    threshold = threshold * 1000;
    if (threshold > 0 && now.getTime() - d.getTime() < threshold) {
      return duration(tdiff(d, now)) + " ago";
    }

    return strftime(fmt || "%x %X", d);
  }
  // }}}


  /***************************************************
    task_id(t) - Truncate a Task UUID, maintaining identifiability

    Task UUIDs are long, too long for humans to be able to differentiate
    or recognize.  `task_id` takes a task object (which has a `uuid` field),
    truncates it to a reasonable length (6-12 chars) and returns it.

   ***************************************************/
  exported.task_id = function (t) { // {{{
    if (t instanceof Object) {
      if (!'uuid' in t) { return "[unknown]" }
      t = t.uuid;
    }
    return t.toString().substr(0,7);
  }
  // }}}


  /***************************************************
    banner(msg, [type]) - Display a small banner message for a limited time.

    Generates and displays a small banner message, useful for notifications,
    status updates, error handling, and more.

    The `msg` argument is required, and contains the text to include in the
    banner itself.  The optional `type` argument specifies a CSS class to
    attach to the banner element for stylistic purposees.  Common `type` values
    include "info" (the default), and "error"

   ***************************************************/
  exported.banner = (function () { // {{{
    var timer = undefined;
    var $banner = null;

    return function (message, type) {
      if (typeof timer !== 'undefined') {
        window.cancelTimeout(timer);
        timer = undefined;
      }

      if (!$banner) { $(document.body).append( $banner = $('<div class="banner-top">').hide() ); }
      if (typeof type == 'undefined') { type = 'info' }
      $banner.show().template('banner', {
        type:    type,
        message: message
      });
      $banner.on('click', 'a', function (event) {
        event.preventDefault();
        $banner.hide();
      });
      if (type !== 'error') {
        time = window.setTimeout(function () {
          $banner.hide();
        }, 7000);
      }
    };
  }());
  // }}}


  /***************************************************
    modal(html) - Display a modal dialog that the user must interact with.

    Display a modal dialog box, which occludes the main page and prevents
    interaction with the rest of the interface.  The `html` argument is the
    complete HTML code (or a DOM object) to place inside the frame of the
    modal dialog.

   ***************************************************/
  exported.modal = (function () { // {{{
    var $wash = $('<div id="modal" class="modal-wash"></div>').hide();
    $(document.body).append($wash);

    $wash.on('click', '.closes, [rel="close"]', function (event) {
      event.preventDefault();
      $wash.hide();
    });

    return function (html) {
      if (html === true) {
        $wash.hide();
        return;
      }
      var $window = $(html);
      $wash.hide().empty().append($window).show();
      return $window;
    }
  }());
  // }}}


  /***************************************************
     api(...) - Make a single AJAX call

   ***************************************************/
  exported.api = (function () { // {{{
    var $key = 10000;
    var $inflight = {};

    return function (options, multi) {
      $key++;
      if (!multi) {
        $.each($inflight, function (i,ajax) {
          ajax.abort();
        });
        $inflight = {};
      }

      if ('data' in options) {
        options.data = JSON.stringify(options.data);
        options.contentType = 'application/json';
      }

      var e = 'An unknown error has occurred.';
      if (typeof(options.error) === 'string') {
        e = options.error;
        delete options.error;
      }

      var complete = options['complete'];
      options.complete = function () {
        delete $inflight[$key];
        if (typeof(complete) !== 'undefined') {
          return complete.apply(this, arguments);
        }
      };

      if (!('error' in options)) {
        options.error = function (xhr) {
          if (xhr.status == 0) {
            return; /* jquery was aborted; no point in erroring... */
          }
          if (xhr.status == 401) {
            document.location.href = '/';
            return
          }
          if (xhr.status == 403) {
            $('#main').template('access-denied', { level: 'generic', need: 'elevated' });
            return
          }
          $('#main').template('error', {
            http:     xhr.status + ' ' + xhr.statusText,
            response: xhr.responseText,
            message:  e,
          });
        };
      }

      $inflight[$key] = $.ajax(options);
      return $inflight[$key];
    };
  })();
  // }}}

  /***************************************************
    website() - Return the base URL of this SHIELD, per document.location
  ***************************************************/
  exported.website = function () { // {{{
    return document.location.toString().replace(/#.*/, '').replace(/\/$/, '');
  };
  // }}}




  /***************************************************
    $(...).serializeObject()

    Given a <form> object, `serializeObject()` will return a simple
    Javascript object (suitable for passing to api()), based on the
    fields present in the form.

    Any fields with dotted names (like config.host) will be expanded
    into multi-level javascript objects, like: { config: { host: x }}

   ***************************************************/
  $.fn.serializeObject = function (flat) { // {{{
    var a = this.serializeArray();
    var o = {};

    if (flat) {
      for (var i = 0; i < a.length; i++) {
        o[a[i].name] = a[i].value;
      }

    } else {
      for (var i = 0; i < a.length; i++) {
        var parts = a[i].name.split(/\./);
        t = o;
        while (parts.length > 1) {
          if (!(parts[0] in t)) { t[parts[0]] = {}; }
          t = t[parts[0]];
          parts.shift();
        }
        t[parts[0]] = a[i].value;
      }
    }

    return o;
  };
  // }}}


  /***************************************************
    autofocus() - Set focus to the first '.autofocus' element

   ***************************************************/
  $.fn.autofocus = function () { // {{{
    var $self = this;
    window.setTimeout(function () {
      $self.find('.autofocus').focus();
    }, 150);
    return $self;
  };
  // }}}


  /***************************************************
    $(...).reset()

    Given a <form> object, `reset()` will reset the form back
    to a pre-validation state, but leave all of the entered
    data intact.

   ***************************************************/
  $.fn.reset = function () { // {{{
    this.find('.error, [data-error]').hide();
    return this;
  };
  // }}}


  /***************************************************
    $(...).missing(lst)

    Given a <form> object, `missing(lst)` will walk the `lst`
    argument, a list of field names, and activate the "missing"
    error for each.  Other errors will be suppressed.

   ***************************************************/
  $.fn.missing = function (fields) { // {{{
    for (var i = 0; i < fields.length; i++) {
      this.error(fields[i], 'missing');
    }
  };
  // }}}


  /***************************************************
    $(...).error(message)
    $(...).error(object)
    $(...).error(field, type)

    Given a <form> object, `error()` shows and hides error
    messages, on a per-field basis, or form-wide.

    If only one argument, a string, is given, it is treated as
    an error message string, and placed in the form-wide error
    container.

    If instead the argument is an object, it will interpret
    the top-level keys thusly:

        "error"    A form-wide error will be issued.
        "missing"  The 'missing' error for each field in the
                   value (a list) will be activated.

    In the two-argument version, the errors will be activated
    on the named field.  This mode only operates on a single
    field at a time.

    In both cases, other errors messages at the same level
    will be suppressed.

   ***************************************************/
  $.fn.error = function () { // {{{
    if (arguments.length == 1 && typeof(arguments[0]) === 'string') {
      this.find('.error').html(arguments[0]).show();

    } else if (arguments.length == 1 && typeof(arguments[0]) === 'object') {
      var what = arguments[0];
      if ('error'   in what) { this.error(what.error);     }
      if ('missing' in what) { this.missing(what.missing); }

    } else if (arguments.length == 2) {
      var what = arguments[1];
      this.find('[data-field="'+arguments[0]+'"] [data-error]').each(function (i, e) {
        var $e = $(e);
        $e.toggle($e.is('[data-error="'+what+'"]'));
      });

    } else {
      throw '$(...).error() given the wrong number of arguments';
    }
  };
  // }}}


  /***************************************************
    $(...).isOK()

    Given a <form> object, `isOK()` returns true if there are
    no visible error messages, and the form can be submitted.

   ***************************************************/
  $.fn.isOK = function () { // {{{
    return this.find('.error:visible, [data-error]:visible').length == 0;
  }; // }}}


  /***************************************************
     $(...).roles(sel) - show a role selection menu

     A new role selection menu element will be created,
     based off of the attributes of the target element,
     and then added into the DOM, positioned absolutely
     so as to appear at the (x,y) coordinate of the top
     left point of the target element (effectively
     obscuring the initiating UI widget).

     The target may have the following data attributes:

      - data-type  Either "system" or "tenant"
      - data-role  The internal name of the pre-selected
                   role.  Valid values are based on the value
                   of data-tyep:

                     system: admin, engineer, or technician
                     tenant: admin, manager, or operator

   ***************************************************/
  $.fn.roles = (function () { /// {{{
    var elem, menu, fn;
    fn = function () { }
    $(document.body).on('click', '.roles-menu', function (event) {
      event.stopPropagation();
      event.preventDefault();

      var $role = $(event.target).closest('div[data-role]');
      if ($role.length == 1) {
        elem.attr('data-role', $role.attr('data-role'));
        elem.html($role.find('strong').html());
        menu.remove();
        fn(elem, $role.attr('data-role'));
        menu = elem = undefined;
        fn = function () { };
      }
    });

    return function (sel, callback) {
      this.on('click', sel, function (event) {
        event.stopPropagation();
        event.preventDefault();

        fn = callback;
        elem = $(event.target);
        var type = elem.extract('type') || 'system';
        var role = elem.extract('role');

        if (menu) { menu.remove(); }
        elem = elem;
        menu = $($.template('roles-menu', {
            type    : type,
            current : role
          })).css({
            display  : 'block',
            position : 'absolute',
            top      : elem.offset().top + 4,
            left     : elem.offset().left + 4
          });
        $(document.body).append(menu);
      });
      return this;
    };
  })();
  // }}}


  /***************************************************
     $(...).userlookup(sel) - wire up a user lookup field

   ***************************************************/
  $.fn.userlookup = function (sel, opts) { // {{{
    opts = $.extend({}, {
      filter: function (l) { return l },
      onclick: function () { }
    }, opts);

    var first = undefined;
    var timer;

    this.on('keydown', sel, function (event) {
      var $field = $(event.target);

      if (event.which == 13) { /* ENTER */
        event.preventDefault();

        if ($('.userlookup-results li').length > 0) {
          opts.onclick(first);
        }
        $('.userlookup-results').remove();
        $field.val('');
      }
    }).on('keyup', sel, function (event) {
      var $field = $(event.target);

      if (timer) { window.clearTimeout(timer) }
      var search = $field.val();
      console.log("search: '%s'", search);
      if (search.length >= 3) {
        timer = window.setTimeout(function () {
          api({
            type: 'POST',
            url:  '/v2/ui/users',
            data: {search: search},
            success: function (data) {
              if (typeof(opts.filter) == 'function') { data = opts.filter(data); }
              first = data[0];

              $('.userlookup-results').remove();
              $(event.target).after(
                $($.template('userlookup-results', data))
                  .on('click', 'li', function (event) {
                    event.preventDefault();
                    event.stopPropagation();

                    opts.onclick(data[$(event.target).extract('idx')]);

                    $('.userlookup-results').remove();
                    $field.val('');
                  })
              );
            }
          });
        }, 400);
      } else {
        console.log('removed');
              $('.userlookup-results').remove();
      }
    });
    return this;
  };
  // }}}


  /***************************************************
     $(...).extract(key) - find and extract data

     This helper wraps up the common idiom:

       $target.closes(['data-x']).data('x')

     To find the nearest ancestor (or self) that has a
     data-something attribute set, and then extract that
     value via jQuery's .data()

   ***************************************************/
  $.fn.extract = function (key) { // {{{
    key = 'data-'+key;
    return this.closest('['+key+']').attr(key);
  };
  // }}}


  /***************************************************
     $(...).transitionClass(class1, class2)

     This helper wraps up the common idiom:

       $target.removeClass('class1').addClass('class2');

     which is a sort of toggleClass() with two classes,
     representing a transition from class1 -> class2.

   ***************************************************/
  $.fn.transitionClass = function (class1, class2) { // {{{
    if (class1) { this.removeClass(class1); }
    if (class2) { this.addClass(class2); }
    return this;
  };
  // }}}


  /***************************************************
     $(...).validate(data) - Validate a Plugin configuration form.

   ***************************************************/
  $.fn.validate = function () { // {{{
    var $form = this;

    $form.find('[data-field]').each(function (i, ctl) {
      var $ctl  = $(ctl);
      var type  = $ctl.attr('data-type');
      var field = $ctl.attr('data-field');
      var value = $ctl.find('[name="'+field+'"]').val() || '';

      console.log('checking field "%s" of type "%s" (value="%s")', field, type, value);

      if ($ctl.is('.required') && value == "") {
        $form.error(field, 'missing');
        console.log('%s is required (value="%s")', field, value);

      } else if (value != "") {
        switch (type) {
        case "pem-x509":
          if (!value.match(/^\s*-----BEGIN CERTIFICATE-----[\s\S]+-----END CERTIFICATE-----/)) {
            $form.error(field, 'invalid');
          }
          break;
        case "pem-rsa-pk":
          if (!value.match(/^\s*-----BEGIN (RSA )?PRIVATE KEY-----[\s\S]+-----END (RSA )?PRIVATE KEY-----/)) {
            $form.error(field, 'invalid');
          }
          break;
        case "abspath":
          console.log(ctl, ' checking value abspath (', value, ')');
          if (!value.match(/^\//)) {
            $form.error(field, 'invalid');
          }
          break;

        case "port":
          console.log(ctl, ' checking value port (', value, ')');
          if (!value.match(/^[0-9]+/)) {
            $form.error(field, 'invalid');
          } else {
            var n = parseInt(value);
            if (n < 1 || n > 65535) {
              $form.error(field, 'invalid');
            }
          }
          break;
        }
      }
    });
    return this;
  };
  // }}}


  /***************************************************
     bytes(x) - Format a file size

     Given an integer number of bytes, formats and returns a more
     human-readable string using units of n*1024b, to make sizes
     more manageable.

   ***************************************************/
  exported.bytes = function (x) { // {{{
    var units = ["b", "K", "M", "G", "T"];
    while (units.length > 1 && x >= 1024) {
      units.shift();
      x /= 1024;
    }
    return (Math.round(x * 10) / 10).toString() + units[0];
  };

  /***************************************************
     $(...).readableToBytes(x) - Format a file size

     Given a human-readable string using units of n*1024b,
     formats and returns an integer number of bytes, to make sizes
     more manageable.

   ***************************************************/
  exported.readableToBytes = function (x) { 
    var powers = {'': 0, 'k': 1, 'm': 2, 'g': 3, 't': 4};
    var regex = /(\d+(?:\.\d+)?)\s?(k|m|g|t)?b?/i;
    var res = regex.exec(x);
    return res[1] * Math.pow(1024, powers[(res[2]||"").toLowerCase()]);
  };
  // }}}


  /***************************************************
     $(...).optgroup() - Set up an optgroup widget

   ***************************************************/
  $.fn.optgroup = function () { // {{{
    this.each(function (i, optgroup) {
      var $optgroup = $(optgroup);
      $optgroup.find('li').on('click', function (event) {
        $optgroup.find('li.selected').removeClass('selected');
        $(event.target).closest('li').addClass('selected');
      });
    });
    return this;
  };
  // }}}


  /***************************************************
     optgroup(selected, list) - Return HTML for an optgroup set

   ***************************************************/
  exported.optgroup = function (selected, list) { // {{{
    if (arguments.length == 1) {
      list = selected;
      selected = list[0];
    }

    var html = "";
    for (var i = 0; i < list.length; i++) {
      if (list[i] == selected) {
        html += '<li class="selected">'+list[i]+'</li>';
      } else {
        html += '<li>'+list[i]+'</li>';
      }
    }
    return html;
  };
  // }}}


  /***************************************************
     $(...).timespec() - Summarize a Timespec Form

   ***************************************************/
  $.fn.timespec = function () { // {{{
    var $form = $(this);
    var d = $form.serializeObject();
    var s = '';

    switch ($form.find('[data-subform].selected').extract('subform')) {
    case 'schedule-hourly':
      var ampm = $form.find('#schedule-hourly .ampm .selected').text();
      return 'every '+d.hourlyn+' hour from '+d.hourlyat+ampm;

    case 'schedule-daily':
      var ampm = $form.find('#schedule-daily .ampm .selected').text();
      return 'daily '+d.dailyat+ampm;

    case 'schedule-weekly':
      var ampm = $form.find('#schedule-weekly .ampm .selected').text();
      var wday = $form.find('#schedule-weekly .wday .selected').text();
      return 'weekly at '+d.weeklyat+ampm+' on '+wday;

    case 'schedule-monthly':
      var ampm = $form.find('#schedule-monthly .ampm .selected').text();
      var mday = $form.find('#schedule-monthly .mday .selected').text();
      d.monthlynth = d.monthlynth.replace(/\s*(th|rd|st)$/, '')+'th';
      if (mday == 'day') {
        return 'monthly at '+d.monthlyat+ampm+' on '+d.monthlynth;
      } else {
        return d.monthlynth+' '+mday+' at '+d.monthlyat+ampm;
      }
    }
  };
  // }}}

  /***************************************************
     $(...).submitting($bool) - Mark a form as "submitting"

   ***************************************************/
  $.fn.submitting = function (on) { // {{{
    this.each(function (i, form) {
      var $form = $(form);
      $form.find('button[data-alt]').each(function (i, b) {
        var $b = $(b);
        if (on) {
          $b.attr('data-alt-orig', $b.html())
            .prop('disabled', true)
            .html($b.attr('data-alt'));
        } else {
          $b.html($b.attr('data-alt-orig'))
            .prop('disabled', false);
        }
      });
    });
    return this;
  };
  // }}}
})(jQuery, window, document);
