;(function (exported, document, undefined) {

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

    return exported.strftime(fmt || "%x %X", d);
  }
  // }}}


  /***************************************************
    strftime(fmt, d) - Format a time, using standard POSIX formatting codes.

    This is the same `strftime()` you know and love from other languages,
    only this one is implemented in Javascript and is missing some of the
    more obscure and arcane formatting codes.

   ***************************************************/
  exported.strftime = function (fmt, d) { // {{{
    d = exported.tparse(d);
    if (typeof(d) === 'undefined') {
      return "";
    }

    en_US = {
      pref: {
        /* %c */ datetime: function (d) { return exported.strftime("%a %b %e %H:%M:%S %Y", d); },
        /* %x */ date:     function (d) { return exported.strftime("%m/%d/%Y", d); },
        /* %X */ time:     function (d) { return exported.strftime("%H:%M:%S", d); }
      },
      weekday: {
        abbr: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'],
        full: ['Sunday',
              'Monday',
              'Tuesday',
              'Wednesday',
              'Thursday',
              'Friday',
              'Saturday']
      },
      month: {
        abbr: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
              'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
        full: ['January',
              'February',
              'March',
              'April',
              'May',
              'June',
              'July',
              'August',
              'September',
              'October',
              'November',
              'December']
      },
      AM: "AM", am: "am", PM: "PM", pm: "pm",
      ordinal: ['th', 'st', 'nd', 'rd', 'th', 'th', 'th', 'th', 'th', 'th', 'th', //  1 - 10
                      'th', 'th', 'th', 'th', 'th', 'th', 'th', 'th', 'th', 'th', // 11 - 20
                      'st', 'nd', 'rd', 'th', 'th', 'th', 'th', 'th', 'th', 'th', // 21 - 30
                      'st'],
      zero:  ['00', '01', '02', '03', '04', '05', '06', '07', '08', '09',
              '10', '11', '12', '13', '14', '15', '16', '17', '18', '19',
              '20', '21', '22', '23', '24', '25', '26', '27', '28', '29',
              '30', '31', '32', '33', '34', '35', '36', '37', '38', '39',
              '40', '41', '42', '43', '44', '45', '46', '47', '48', '49',
              '50', '51', '52', '53', '54', '55', '56', '57', '58', '59'],

      space: [' 0', ' 1', ' 2', ' 3', ' 4', ' 5', ' 6', ' 7', ' 8', ' 9',
              '10', '11', '12', '13', '14', '15', '16', '17', '18', '19',
              '20', '21', '22', '23', '24', '25', '26', '27', '28', '29',
              '30', '31', '32', '33', '34', '35', '36', '37', '38', '39',
              '40', '41', '42', '43', '44', '45', '46', '47', '48', '49',
              '50', '51', '52', '53', '54', '55', '56', '57', '58', '59'],
    };

    var lc = en_US;

    var inspec = false;
    var alt_o  = false;

    var s = '';
    for (var i = 0; i < fmt.length; i++) {
      var c = fmt.charCodeAt(i);
      if (inspec) {
        switch (c) {
        // %%   A literal '%' character
        case 37:
          s += '%';
          break;

        // %a   The abbreviated name of the day of the week according to the
        //      current locale.
        case 97:
          s += lc.weekday.abbr[d.getDay()];
          break;

        // %A   The full name of the day of the week according to the current
        //      locale.
        case 65:
          s += lc.weekday.full[d.getDay()];
          break;

        // %b   The abbreviated month name according to the current locale.
        case 98:
          s += lc.month.abbr[d.getMonth()];
          break;

        // %h   Equivalent to %b.
        case 104:
          s += lc.month.abbr[d.getMonth()];
          break;

        // %B   The full month name according to the current locale.
        case 66:
          s += lc.month.full[d.getMonth()];
          break;

        // %c   The preferred date and time representation for the current
        //      locale.
        case 99:
          s += lc.pref.datetime(d);
          break;

        // %C   The century number (year/100) as a 2-digit integer
        case 67:
          s += d.getFullYear() / 100;
          break;

        // %d   The day of the month as a decimal number (range 01 to 31).
        case 100:
          s += lc.zero[d.getDate()];
          break;

        // %D   Equivalent to %m/%d/%y.  (Yecch—for Americans only.  Americans
        //      should note that in other countries %d/%m/%y is rather common.
        //      This means that in international context this format is
        //      ambiguous and should not be used.)
        case 68:
          s += exported.strftime("%m/%d/%y", d);
          break;

        // %e   Like %d, the day of the month as a decimal number, but a
        //      leading zero is replaced by a space.
        case 101:
          s += d.getDate().toString()+(alt_o ? lc.ordinal[d.getDate()] : '');
          break;

        // %E   Modifier: use alternative format, see below.
        case 69:
          // not supported; just skip it
          continue;

        // %F   Equivalent to %Y-%m-%d (the ISO 8601 date format).
        case 70:
          s += exported.strftime("%Y-%m-%d", d);
          break;

        // %G   The ISO 8601 week-based year (see NOTES) with century as a
        //      decimal number.  The 4-digit year corresponding to the ISO
        //      week number (see %V).  This has the same format and value as
        //      %Y, except that if the ISO week number belongs to the previous
        //      or next year, that year is used instead.
        case 71:
          throw "this strftime() does not support '%G'"; // FIXME

        // %g   Like %G, but without century, that is, with a 2-digit year
        //      (00-99).
        case 103:
          throw "this strftime() does not support '%g'"; // FIXME

        // %H   The hour as a decimal number using a 24-hour clock (range 00 to 23).
        case 72:
          s += lc.zero[d.getHours()]
          break;

        // %I   The hour as a decimal number using a 12-hour clock (range 01 to 12)
        case 73:
          s += lc.zero[d.getHours() % 12 == 0 ? 12 : d.getHours() % 12];
          break;

        // %j   The day of the year as a decimal number (range 001 to 366).
        case 106:
          throw "this strftime() does not support '%j'"; // FIXME

        // %k   The hour (24-hour clock) as a decimal number (range 0 to 23);
        //      single digits are preceded by a blank.  (See also %H.)
        case 107:
          s += lc.space[d.getHours()];
          break;

        // %l   The hour (12-hour clock) as a decimal number (range 1 to 12);
        //      single digits are preceded by a blank.  (See also %I.)
        case 108:
          s += lc.space[d.getHours() % 12 == 0 ? 12 : d.getHours() % 12];
          break;

        // %m   The month as a decimal number (range 01 to 12).
        case 109:
          s += lc.zero[d.getMonth()+1];
          break;

        // %M   The minute as a decimal number (range 00 to 59).
        case 77:
          s += lc.zero[d.getMinutes()];
          break;

        // %n   A newline character.
        case 110:
          s += "\n";
          break;

        // %O   Modifier: use alternative format, see below.
        case 79:
          alt_o = true;
          continue;

        // %p   Either "AM" or "PM" according to the given time value, or the
        //      corresponding strings for the current locale.  Noon is treated
        //      as "PM" and midnight as "AM".
        case 112:
          s += (d.getHours() < 12 ? lc.AM : lc.PM);
          break;

        // %P   Like %p but in lowercase: "am" or "pm" or a corresponding
        //      string for the current locale.
        case 80:
          s += (d.getHours() < 12 ? lc.am : lc.pm);
          break;

        // %r   The time in a.m. or p.m. notation.  In the POSIX locale this
        //      is equivalent to %I:%M:%S %p.
        case 114:
          s += lc.zero[d.getHours() % 12 == 0 ? 12 : d.getHours() % 12] + ":" +
              lc.zero[d.getMinutes()]                                  + ":" +
              lc.zero[d.getSeconds()]                                  + " " +
              (d.getHours() < 12 ? lc.AM : lc.PM);
          break;

        // %R   The time in 24-hour notation (%H:%M).  For a version
        //      including the seconds, see %T below.
        case 82:
          s += lc.zero[d.getHours()] + ":" +
              lc.zero[d.getMinutes()];
          break;

        // %s   The number of seconds since the Epoch,
        //      1970-01-01 00:00:00+0000 (UTC).
        case 115:
          s += d.getTime().toString();
          break;

        // %S   The second as a decimal number (range 00 to 60).  (The range
        //      is up to 60 to allow for occasional leap seconds.)
        case 83:
          s += lc.zero[d.getSeconds()];
          break;

        // %t   A tab character.
        case 116:
          s += "\t";
          break;

        // %T   The time in 24-hour notation (%H:%M:%S).
        case 84:
          s += lc.zero[d.getHours()]   + ":" +
              lc.zero[d.getMinutes()] + ":" +
              lc.zero[d.getSeconds()];
          break;

        // %u   The day of the week as a decimal, range 1 to 7, Monday being 1.
        //       See also %w.
        case 117:
          s += (d.getDay()+1).toString()+(alt_o ? lc.ordinal[d.getDay()+1] : '');
          break;

        // %U   The week number of the current year as a decimal number, range
        //      00 to 53, starting with the first Sunday as the first day of
        //      week 01.  See also %V and %W.
        case 85:
          throw "this strftime() does not support '%U'"; // FIXME

        // %V   The ISO 8601 week number (see NOTES) of the current year as a
        //      decimal number, range 01 to 53, where week 1 is the first week
        //      that has at least 4 days in the new year.  See also %U and %W.
        case 86:
          throw "this strftime() does not support '%V'"; // FIXME

        // %w   The day of the week as a decimal, range 0 to 6, Sunday being 0
        //      See also %u.
        case 119:
          s += (d.getDay()).toString();
          break;

        // %W   The week number of the current year as a decimal number, range
        //      00 to 53, starting with the first Monday as the first day of
        //      week 01.
        case 87:
          throw "this strftime() does not support '%W'"; // FIXME

        // %x   The preferred date representation for the current locale
        //      without the time.
        case 120:
          s += lc.pref.date(d);
          break;

        // %X   The preferred time representation for the current locale
        //      without the date.
        case 88:
          s += lc.pref.time(d);
          break;

        // %y   The year as a decimal number without a century (range 00 to 99).
        case 121:
          s += lc.zero[d.getFullYear() % 100];
          break;

        // %Y   The year as a decimal number including the century.
        case 89:
          s += d.getFullYear();
          break;

        // %z   The +hhmm or -hhmm numeric timezone (that is, the hour and
        //      minute offset from UTC).
        case 122:
          throw "this strftime() does not support '%z'"; // FIXME

        // %Z   The timezone name or abbreviation.
        case 90:
          throw "this strftime() does not support '%Z'"; // FIXME

        default:
          throw "unrecognized strftime sequence '%"+fmt[i]+"'";
        }

        inspec = false;
        alt_o  = false;
        continue;
      }

      if (c == 37) { // %
        inspec = true
        continue;
      }

      s += fmt[i];
    }
    return s;
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
      $banner.show().html(exported.template('banner', {
        type:    type,
        message: message
      }));
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
    template(name, data) - Render a dynamic page template, with the given data.

    Templates are expected to be stored in-DOM, inside of <script> tags with
    ID attributes in the form "tpl--$name".  Raw templates will be compiled into
    executable code as needed, and cached for future use, since template contents
    are not expected to change.

    The template language is javascript, embedded in HTML:

      [[= expr ]]   Evaluate expression and concatenate the result
                    to the output string.

      [[ stmt ]]    Execute stmt; if it produces a value, discard
                    the value.  Useful for embedding control-flow
                    into the template.

   ***************************************************/
  exported.template = (function () { // {{{

    /* compile a template <script> into a function */
    var compile = function (src) {
      var tokenizer = new RegExp('([\\s\\S]*?)\\[\\[([\\s\\S]*?)\\]\\]([\\s\\S]*)');
      var str = function (s) {
        if (!s) { return "''"; }
        return "'"+s.replace(/(['\\])/g, '\\$1').replace(/\n/g, "\\n")+"'";
      };

      var code = [];
      for (;;) {
        var tokens = tokenizer.exec(src)
        if (!tokens) {
          code.push('__ += '+str(src)+';');
          break;
        }
        code.push('__ += '+str(tokens[1])+';');
        if (tokens[2][0] == '=') {
          code.push('__ += ('+tokens[2].replace(/^=\s*/, '')+');');
        } else if (tokens[2][0] != '#') { /* skip comments */
          code.push(tokens[2]);
        }
        src = tokens[3];
      }
      code = code.join('');
      return function (_) {
        var __ = '';
        var maybe = function (a,b) { return typeof(a) !== 'undefined' ? a : b; };
        var html  = function (s) {
          return $('<textarea>').text(s).html()
            .replace(/&lt;(https?:.+?)&gt;/g, '<a target="_blank" href="$1">$1</a>')
            .replace(/\n/g, '<br>'); };

        eval(code);
        return __;
      };
    }

    var Templates = {};
    return function (name, data) {
      if (!(name in Templates)) {
        console.log('compiling template %s', name);
        Templates[name] = compile($('script#tpl--'+name).html());
      }
      console.log('rendering template %s, with data:', name, data);
      return Templates[name](data);
    }
  })();
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
            $('#main').html(template('access-denied', { level: 'generic', need: 'elevated' }));
            return
          }
          $('#main').html(exported.template('error', {
            http:     xhr.status + ' ' + xhr.statusText,
            response: xhr.responseText,
            message:  e,
          }));
        };
      }

      $inflight[$key] = $.ajax(options);
      return $inflight[$key];
    };
  })();
  // }}}

  /***************************************************
     apis(...) - Combine multiple AJAX calls

   ***************************************************/
  exported.apis = (function () { // {{{
    return function (options) {
      var failed = false;
      var nwait = 0;
      var data = {};

      var e = 'An unknown error has occurred.';
      if (typeof(options.error) === 'string') {
        e = options.error;
        delete options.error;
      }

      if (!('error' in options)) {
        options.error = function (xhr) {
          if (xhr.status == 0) {
            return; /* jquery was aborted; no point in erroring... */
          }
          $('#main').html(exported.template('error', {
            http:     xhr.status + ' ' + xhr.statusText,
            response: xhr.responseText,
            message:  e,
          }));
        };
      }

      var done = function () {
        if (failed) {
          options.error(failed); /* failed is an xhr */
        } else {
          options.success(data);
        }
      };

      for (var key in options.multiplex) {
        (function (k) {
          nwait++;
          api({
            type:  options.multiplex[k].type,
            url:   options.multiplex[k].url.replace(/^\+/, options.base),
            data:  options.multiplex[k].data,
            error:    function (xhr) { failed = xhr; },
            success:  function (dat) { data[k] = dat; },
            complete: function () {
              nwait--; if (nwait <= 0) { done() }
            }
          }, nwait > 1);
        })(key);
      }
    };
  })();
  // }}}


  /***************************************************
    website() - Return the base URL of this SHIELD, per document.location

   ***************************************************/
  exported.website = function () { // {{{
    return document.location.toString().replace(/#.*/, '').replace(/\/$/, '');
  }
  // }}}


  /***************************************************
    $(...).serializeObject()

    Given a <form> object, `serializeObject()` will return a simple
    Javascript object (suitable for passing to api()), based on the
    fields present in the form.

    Any fields with dotted names (like config.host) will be expanded
    into multi-level javascript objects, like: { config: { host: x }}

   ***************************************************/
  exported.jQuery.fn.serializeObject = function () { // {{{
    var a = this.serializeArray();
    var o = {};
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
    return o;
  };
  // }}}


  /***************************************************
    autofocus() - Set focus to the first '.autofocus' element

   ***************************************************/
  exported.jQuery.fn.autofocus = function () { // {{{
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
  exported.jQuery.fn.reset = function () { // {{{
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
  exported.jQuery.fn.missing = function (fields) { // {{{
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
  exported.jQuery.fn.error = function () { // {{{
    if (arguments.length == 1 && typeof(arguments[0]) === 'string') {
      this.find('.error').html(arguments[0]).show();

    } else if (arguments.length == 1 && typeof(arguments[0]) === 'object') {
      var what = arguments[0];
      if ('error'   in what) { this.error(what.error);     }
      if ('missing' in what) { this.missing(what.missing); }

    } else if (arguments.length == 2) {
      var what = arguments[1];
      this.find('.ctl[data-field="'+arguments[0]+'"] [data-error]').each(function (i, e) {
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
  exported.jQuery.fn.isOK = function () { // {{{
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
  exported.jQuery.fn.roles = (function () { /// {{{
    var elem, menu, fn;
    fn = function () { }
    $(document.body).on('click', '.roles-menu', function (event) {
      event.stopPropagation();
      event.preventDefault();

      var $role = $(event.target).closest('div[data-role]');
      if ($role.length == 1) {
        elem.data('role', $role.data('role'));
        elem.html($role.find('strong').html());
        menu.remove();
        fn(elem, $role.data('role'));
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
        menu = $(exported.template('roles-menu', {
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
  exported.jQuery.fn.userlookup = function (sel, opts) { // {{{
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
                $(exported.template('userlookup-results', data))
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
  exported.jQuery.fn.extract = function (key) { // {{{
    return this.closest('[data-'+key+']').data(key);
  };
  // }}}


  /***************************************************
     $(...).validate(data) - Validate a Plugin configuration form.

   ***************************************************/
  exported.jQuery.fn.validate = function (data) { // {{{
    var $form = this;
    if (!('config' in data)) { data.config = {}; }

    $form.find('.ctl').each(function (i, ctl) {
      var $ctl  = $(ctl);
      var type  = $ctl.data('type');
      var field = $ctl.data('field');
      var key   = field.replace(/^config\./, '');

      if ($ctl.is('.required') && !(key in data.config)) {
        $form.error(field, 'missing');

      } else if (key in data.config) {
        switch (type) {
        case "abspath":
          if (!data.config[key].match(/^\//)) {
            $form.error(field, 'invalid');
          }
          break;

        case "port":
          if (!data.config[key].match(/^[0-9]+/)) {
            $form.error(field, 'invalid');
          } else {
            var n = parseInt(data.config[key]);
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
     $(...).subform() - Set up a subform widget

   ***************************************************/
  exported.jQuery.fn.subform = function () { // {{{
    var $form = this;

    $form.find('.subform').hide();
    $form.find('[data-subform]').on('click', function (event) {
      $form.find('.subform').hide();
      $form.find('.subform#'+$(event.target).extract('subform')).show();
    });

    return this;
  };
  // }}}


  /***************************************************
     $(...).optgroup() - Set up an optgroup widget

   ***************************************************/
  exported.jQuery.fn.optgroup = function () { // {{{
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
  exported.jQuery.fn.timespec = function () { // {{{
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
     $(...).pluginForm() - Plugin Form UI Widget

   ***************************************************/
  exported.jQuery.fn.pluginForm = function (opts) { // {{{
    opts = $.extend({}, {}, opts);
    if (!('type' in opts)) {
      throw 'pluginForm() requires a "type" option (either "store" or "target")';
    }

    var $parent = this;
    var cache   = {};

    $parent
      .on('change', 'select[name=agent]', function (event) {
        var uuid = $(event.target).val().replace(/\|.*/, '');
        if (uuid == "") { return; }

        $parent.find('#choose-plugin').html(template('loading'));
        api({
          type: 'GET',
          url:  '/v2/tenants/'+$global.auth.tenant.uuid+'/agents/'+uuid,
          success: function (data) {
            cache = data;
            data.type = opts.type;
            $parent.find('#choose-plugin').html(template('subform-choose-plugin', data));
            if (opts.plugin) {
              $parent.find('[name=plugin]').val(opts.plugin).trigger('change');
            }
          }
        });
      })
      .on('change', 'select[name=plugin]', function (event) {
        var plugin = $(event.target).val();
        if (plugin == '') { return; }

        $parent.find('#configure-plugin').html(
          template('subform-configure-plugin', {
            type:   opts.type,
            plugin: cache.metadata.plugins[plugin],
            config: (plugin == opts.plugin ? opts.config : undefined)
          }));
      });
  };
  // }}}


  /***************************************************
     $(...).serializePluginObject() - Serialize Data from a Plugin Form

   ***************************************************/
  exported.jQuery.fn.serializePluginObject = function () { // {{{
    var data       = this.serializeObject();
    data.agent     = data.agent.replace(/.*\|/, '');
    if ('threshold' in data) {
      data.threshold = readableToBytes(data.threshold || '0');
    }
    if (!('config' in data)) {
      data.config = {};
    }
    for (var k in data.config) {
      if (data.config[k] == "") {
        delete data.config[k];
      }
    }

    return data;
  };
  // }}}


  exported.watchTasks = (function () {
    var socket = undefined;

    return function (tenant, target, cb) {
      if (socket) { socket.close(); }

      console.log("watching tasks for tenant "+tenant+", target "+target)
      socket = new WebSocket(document.location.protocol.replace(/http/, 'ws')+"//"+document.location.host+"/v2/tenants/"+tenant+"/systems/"+target+"/events");
      socket.onmessage = cb;
    }
  })();

  exported.sockify = function (tenant) {
    var socket = new WebSocket(document.location.protocol.replace(/http/, 'ws')+"//"+document.location.host+"/v2/tenants/"+tenant+"/events");
    socket.onopen    = function () { console.log('socket open: ',    arguments); };
    socket.onclose   = function () { console.log('socket close: ',   arguments); };
    socket.onmessage = function () { console.log('socket message: ', arguments);
      console.log(arguments[0].message.data);
      console.dir(JSON.parse(arguments[0].message.data));
    };
    return socket;
  };

  /***************************************************
     $(...).serializePluginObject() - Serialize Data from a Plugin Form

   ***************************************************/
  exported.viewSwitcher = function() {
   $(document.body).on('click', '.switch-me .switcher a[href^="switch:"]', function (event) {
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
    });
  };
})(window, document);
