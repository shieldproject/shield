var Lens = {};
;(function () {

  var nil = function () { };

  if (!console) {
    console = { log: nil };
  }

  var log = {
    debug: nil,
    info:  nil,
    warn:  nil,
    error: nil,

    level: function (level) {
      log.debug = nil;
      log.info  = nil;
      log.warn  = nil;
      log.error = nil;

      switch (level) {
      case 'debug': log.debug = console.log;
      case 'info':  log.info  = console.log;
      case 'warn':  log.warn  = console.log;
      case 'error': log.error = console.log;
      }
    }
  };

  Lens.log = log;
})()
;(function () {
  const SET_ATTRIBUTE    = 1,
        REMOVE_ATTRIBUTE = 2,
        SET_PROPERTY     = 3,
        REMOVE_PROPERTY  = 4,
        REPLACE_NODE     = 5,
        APPEND_CHILD     = 6,
        REMOVE_CHILD     = 7,
        REPLACE_TEXT     = 8;

  /* diff the ATTRIBUTES of two nodes, and return a
     (potentially empty) patch op list. */
  var diffa = function (a, b) {
    const ops = [];
    const _a  = {};
    const _b  = {};

    for (let i = 0; i < a.attributes.length; i++) {
      _a[a.attributes[i].nodeName] = a.attributes[i].nodeValue;
    }
    for (let i = 0; i < b.attributes.length; i++) {
      _b[b.attributes[i].nodeName] = b.attributes[i].nodeValue;
    }

    /* if the attribute is only defined in (a), then
       it has been removed in (b) and should be patched
       as a REMOVE_ATTRIBUTE.  */
    for (let attr in _a) {
      if (!(attr in _b)) {
        ops.push({
          op:    REMOVE_ATTRIBUTE,
          node:  a,
          key:   attr,
        });
      }
    }

    /* if the attribute is only defined in (b), or is
       defined in both with different values, patch as
       a SET_ATTRIBUTE to get the correct value. */
    for (let attr in _b) {
      if (!(attr in _a) || _a[attr] !== _b[attr]) {
        ops.push({
          op:    SET_ATTRIBUTE,
          node:  a,
          key:   attr,
          value: _b[attr]
        });
      }
    }

    return ops;
  };

  var diffe = function (a, b) {
    if (a.localName === b.localName) {
      return diffa(a, b);
    }
    return null;
  };

  var difft = function (a, b) {
    if (a.textContent === b.textContent) {
      return [];
    }
    return [{
      op:   REPLACE_TEXT,
      node: a,
      with: b.textContent
    }];
  };

  /* diff two NODEs, without recursing through child nodes */
  var diffn1 = function (a, b) {
    if (a.nodeType != b.nodeType) {
      /* nothing in common */
      return null;
    }

    if (a.nodeType == Node.ELEMENT_NODE) {
      return diffe(a, b);
    }
    if (a.nodeType == Node.TEXT_NODE) {
      return difft(a, b);
    }
    if (a.nodeType == Node.COMMENT_NODE) {
      return null;
    }

    console.log('unrecognized a type %s', a.nodeType);
    return null;
  };

  /* diff two NODEs, co-recursively with diff() */
  var diffn = function (a, b) {
    let ops = diffn1(a, b);

    if (ops) {
      return ops.concat(diff(a, b));
    }

    return [{
      op:   REPLACE_NODE,
      node: a,
      with: b
    }];
  };

  window.diff = function (a, b) {
    let ops = [];
    const { childNodes: _a } = a;
    const { childNodes: _b } = b;

    const _al = _a ? _a.length : 0;
    const _bl = _b ? _b.length : 0;

    for (let i = 0; i < _bl; i++) {
      if (!_a[i]) {
        ops.push({
          op:    APPEND_CHILD,
          node:  a,
          child: _b[i]
        });
        continue;
      }

      ops = ops.concat(diffn(_a[i], _b[i]));
    }

    for (var i = _bl; i < _al; i++) {
      ops.push({
        op:    REMOVE_CHILD,
        node:  a,
        child: _a[i]
      });
    }

    return ops;
  };




  window.patch = function (e, ops) {
    for (let i = 0; i < ops.length; i++) {
      switch (ops[i].op) {
        case SET_ATTRIBUTE:    ops[i].node.setAttribute(ops[i].key, ops[i].value);            break;
        case REMOVE_ATTRIBUTE: ops[i].node.removeAttribute(ops[i].key);                       break;
        case SET_PROPERTY:     /* FIXME needs implemented! */                                 break;
        case REMOVE_PROPERTY:  /* FIXME needs implemented! */                                 break;
        case REPLACE_NODE:     ops[i].node.parentNode.replaceChild(ops[i].with, ops[i].node); break;
        case APPEND_CHILD:     ops[i].node.appendChild(ops[i].child);                         break;
        case REMOVE_CHILD:     ops[i].node.removeChild(ops[i].child);                         break;
        case REPLACE_TEXT:     ops[i].node.textContent = ops[i].with;                         break;
        default:
          console.log('unrecognized patch op %d for ', ops[i].op, ops[i]);
          break;
      }
    }
  };




  window.explainPatch = function (ops) {
    var l = [];
    for (let i = 0; i < ops.length; i++) {
      switch (ops[i].op) {
        case SET_ATTRIBUTE:    l.push(['SET_ATTRIBUTE',    ops[i].node, ops[i].key+'='+ops[i].value]); break;
        case REMOVE_ATTRIBUTE: l.push(['REMOVE_ATTRIBUTE', ops[i].node, ops[i].key]);                  break;
        case SET_PROPERTY:     l.push(['SET_PROPERTY',     'FIXME']);                                  break;
        case REMOVE_PROPERTY:  l.push(['REMOVE_PROPERTY',  'FIXME']);                                  break;
        case REPLACE_NODE:     l.push(['REPLACE_NODE',     ops[i].node, { with: ops[i].with }]);       break;
        case APPEND_CHILD:     l.push(['APPEND_CHILD',     ops[i].node, { child: ops[i].child }]);     break;
        case REMOVE_CHILD:     l.push(['REMOVE_CHILD',     ops[i].node, { child: ops[i].child }]);     break;
        case REPLACE_TEXT:     l.push(['REPLACE_TEXT',     ops[i].node, { with: ops[i].with }]);       break;
        default:               l.push(['**UNKNOWN**',      ops[i]]);                                   break;
      }
    }
    return l;
  };
})(window, document);
;(function () {

  var __templates = {};
  var template = function (name, data) {
    if (!(name in __templates)) {
      Lens.log.debug('template {%s} not found in the cache; compiling from source.', name);
      __templates[name] = compile(name);
    }

    return __templates[name](data || {});
  };

  var parse = function (src) {
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

      if (tokens[2][0] == ':') { /* trim preceeding literal */
        tokens[1] = tokens[1].replace(/\s+$/, '');
        tokens[2] = tokens[2].substr(1);
      }
      if (tokens[2][tokens[2].length - 1] == ':') { /* trim following literal */
        tokens[3] = tokens[3].replace(/^\s+/, '');
        tokens[2] = tokens[2].substr(0, tokens[2].length-2);
      }

      code.push('__ += '+str(tokens[1])+';');
      if (tokens[2][0] == '=') {
        code.push('__ += ('+tokens[2].replace(/^=\s*/, '')+');');

      } else if (tokens[2][0] != '#') { /* skip comments */
        code.push(tokens[2]);
      }

      src = tokens[3];
    }

    return code.join('');
  };

  var compile = function (name) {
    name = name.toString();
    var script = document.getElementById('template:'+name);
    if (!script) {
      Lens.log.error('unable to find a <script> element with id="template:%s"', name);
      return function () {
        throw "Template {"+name+"} not found";
      };
    }

    var code = parse(script.innerHTML);

    return function (_) {
      /* the output variable */
      var __ = '';

      /* namespaced helper functions */
      var lens = {



        /* maybe(x,fallback)

           fallback to a default value if a given variable
           is undefined, or was not provided.

           example:

             [[= lens.maybe(x, "no x given") ]]

         */
        maybe: function (a, b) {
          return typeof(a) !== 'undefined' ? a : b;
        },

        /* escapeHTML(x)

           return a sanitized version of x, with the dangerous HTML
           entities like <, > and & replaced.  Also replaces double
           quote (") with the &quot; representation, so that you can
           embed values in form element attributes.

           example:

             <input type="text" name="display"
                    value="[[= lens.htmlEscape(_.display) ]]">

           lens.h() is an alias, so you can also do this:

             <input type="text" name="display"
                    value="[[= lens.h(_.display) ]]">
         */
        escapeHTML: function (s) {
          var t = document.createElement('textarea');
          t.innerText = s;
          return t.innerHTML.replace(/"/g, '&quot;');
        },

        /* include(template)
           include(template, _.other.data)

           splices the output of another template into the current
           output, at the calling site.  This can be useful for
           breaking up common elements of a UI into more manageable
           chunks.

           example:

             <div id="login">[[ lens.include('signin'); ]]</div>

           you can also provide a data object that will become the
           `_` variable inside the called template:

             [[ lens.include('alert', { alert: "something broke" }); ]]

           as a "language construct", this function is also aliased
           as (toplevel) `include()`, and it can be used in [[= ]]
           constructs:

             [[= include('other-template') ]]

         */
        include: function (name, data) {
          __ += template(name, data || _);
          return '';
        }
      };

      /* aliases ... */
      lens.u = encodeURIComponent;
      lens.h = lens.escapeHTML;
      var include = lens.include;

      Lens.log.debug('evaluating the {%s} template', name);
      eval(code);
      return __;
    };
  };

  window.parseTemplate = parse;

  if (typeof(jQuery) !== 'undefined') {
    jQuery.template = template;

    jQuery.fn.template = function (name, data, force) {
      if (force || this.length == 0) {
        this.html(template(name, data)).data('lens:template', {
          template: name,
          data:     data
        });
        return this;
      }

      var was = this.data('lens:template');
      if (was && (!name || was.template == name)) {
        data = typeof(data) === 'undefined' ? was.data : data;
        window.patch(this[0], diff(this[0], $('<div>'+template(was.template, data)+'</div>')[0]));
        this.data('lens:template', {
          template: was.template,
          data:     data
        });
        return this;
      }

      this.html(template(name, data)).data('lens:template', {
        template: name,
        data:     data
      });
      return this;
    };

  } else if (typeof(window) !== 'undefined') {
    window.template = template;

  } else {
    throw 'neither jQuery or top-level window object were found; unsure where to attach template()...';
  }
})();
;(function () {
  var strftime = function (fmt, d) {
    if (!(d instanceof Date)) {
      var _d = new Date();
      if (!isNaN(d)) {
        _d.setTime(d * 1000); /* epoch s -> ms */
      }
      d = _d;
    }
    if (typeof(d) === 'undefined') {
      return "";
    }

    en_US = {
      pref: {
        /* %c */ datetime: function (d) { return strftime("%a %b %e %H:%M:%S %Y", d); },
        /* %x */ date:     function (d) { return strftime("%m/%d/%Y", d); },
        /* %X */ time:     function (d) { return strftime("%H:%M:%S", d); }
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
          s += parseInt(d.getFullYear() / 100);
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
          s += strftime("%m/%d/%y", d);
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
          s += strftime("%Y-%m-%d", d);
          break;

        // %G   The ISO 8601 week-based year (see NOTES) with century as a
        //      decimal number.  The 4-digit year corresponding to the ISO
        //      week number (see %V).  This has the same format and value as
        //      %Y, except that if the ISO week number belongs to the previous
        //      or next year, that year is used instead.
        case 71:
          throw "this strftime() does not support '%G'";

        // %g   Like %G, but without century, that is, with a 2-digit year
        //      (00-99).
        case 103:
          throw "this strftime() does not support '%g'";

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
          throw "this strftime() does not support '%j'";

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
          var wday = d.getDay();
          if (wday == 0) { wday = 7 };
          s += (wday).toString()+(alt_o ? lc.ordinal[wday] : '');
          break;

        // %U   The week number of the current year as a decimal number, range
        //      00 to 53, starting with the first Sunday as the first day of
        //      week 01.  See also %V and %W.
        case 85:
          throw "this strftime() does not support '%U'";

        // %V   The ISO 8601 week number (see NOTES) of the current year as a
        //      decimal number, range 01 to 53, where week 1 is the first week
        //      that has at least 4 days in the new year.  See also %U and %W.
        case 86:
          throw "this strftime() does not support '%V'";

        // %w   The day of the week as a decimal, range 0 to 6, Sunday being 0
        //      See also %u.
        case 119:
          s += (d.getDay()).toString();
          break;

        // %W   The week number of the current year as a decimal number, range
        //      00 to 53, starting with the first Monday as the first day of
        //      week 01.
        case 87:
          throw "this strftime() does not support '%W'";

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
          throw "this strftime() does not support '%z'";

        // %Z   The timezone name or abbreviation.
        case 90:
          throw "this strftime() does not support '%Z'";

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
  };

  Lens.strftime = strftime;
  window.strftime = strftime;
})();
