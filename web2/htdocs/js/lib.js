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
    var now = new Date()

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
      $banner.show('fade').html(template('banner', {
        type:    type,
        message: message
      }));
      time = window.setTimeout(function () {
        $banner.hide('fade');
      }, 7000);
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

    $wash.on('click', '[rel="close"]', function (event) {
      event.preventDefault();
      $wash.hide();
    });

    return function (html) {
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
        Templates[name] = compile($('script#tpl--'+name).html());
      }
      return Templates[name](data);
    }
  })();
  // }}}


  /***************************************************

   ***************************************************/
  exported.api = (function () { // {{{
    return function (options) {
      if ('data' in options) {
        options.data = JSON.stringify(options.data);
        options.contentType = 'application/json';
      }

      var e = 'An unknown error has occurred.';
      if (typeof(options.error) === 'string') {
        e = options.error;
        delete options.error;
      }

      if (!('error' in options)) {
        options.error = function (xhr) {
          $('#main').html(exported.template('error', {
            http:     xhr.status + ' ' + xhr.statusText,
            response: xhr.responseText,
            message:  e,
          }));
        };
      }

      return $.ajax(options);
    };
  })();
  // }}}
})(window, document);
