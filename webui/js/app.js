(function(window, document, undefined) {

  function notify(level, html) {
    $('#notif').append('<div class="bg-'+level+'">'+html+'</div>');
  }

  var ajax = (function () {
    var AJAX = null;
    return function(params) {
      if (AJAX) { AJAX.abort() }
      AJAX = $.ajax(params)
        .always(function() { AJAX = null })
        .fail(function(jqxhr) {
          // reload the browser if we had an ajax call needing oauth, as cors will prevent this + cause errors
          if (jqxhr.status == 401 && jqxhr.getResponseHeader("WWW-Authenticate").toLowerCase().StartsWith == "bearer") {
            location.reload(true)
          }
          notify('danger', "Backend request failed: "+jqxhr.status.toString()+' '+jqxhr.statusText);
        });
    };
  })();

  var noop = function() { };

  var lastly = function (cc) {
    return function (event) {
      event.stopImmediatePropagation();
      event.preventDefault();
      cc.apply(this, arguments);
    }
  };

  var rejson = function(x) {
    try {
      return JSON.stringify(JSON.parse(x), null, 2);
    } catch (e) {
      return x;
    }
  };

  $.fn.serializeObject = function() {
    a = this.serializeArray();
    o = {};
    for (var i = 0; i < a.length; i++) {
      o[a[i].name] = a[i].value;
    }
    return o;
  };

  /*************************************************************/

  var LOCAL = {}; // local web UI state
  var DB = {};    // saved data from queries

  function indexby(type, id) {
    /* do we have the data to index? */
    if (!(type in DB)) { return {} }

    var idx = {};
    for (var i = 0; i < DB[type].length; i++) {
      if (!(id in DB[type][i])) {
        console.warn("%s[%d] is missing the %s indexing attr: %v", type, i, id, DB[type][i]);
        continue;
      }
      idx[DB[type][i][id]] = DB[type][i];
    }
    return idx;
  }

  var MODEL = (function () {
    var buster = 1;

    return {
      lister: function (url, key) {
        return function () {
          var opts = { cached: true };
          var cc = arguments[0];

          if (!(arguments[0] instanceof Function)) {
            opts = arguments[0];
            cc   = arguments[1];
          }

          if (opts.cached && (key in DB) && DB[key]) {
            cc.apply(null, [key]);
            return;
          }
          ajax({
            type: 'GET',
            url:  url,
            success: function(l) {
              DB[key] = l;
              cc.apply(null, [key]);
            }
          });
        };
      },

      searcher: function (url, key) {
        return function (q, cc) {
          qs = '';
          for (k in q) {
            qs  += '&' + encodeURIComponent(k) + '=' + encodeURIComponent(q[k]);
          }

          ajax({
            type: 'GET',
            url:  url + '?_=' + (buster++) + qs,
            success: function (l) {
              cc.apply(null, [key, l]);
            }
          });
        };
      },

      findOne: function (url, fixup) {
        return function (id, cc) {
          ajax({
            type: 'GET',
            url:  url.replace(/%s/, id),
            success: function (data) {
              if (fixup instanceof Function) {
                data = fixup(data);
              }
              cc.apply(null, [data]);
            }
          });
        };
      },

      creater: function (url, fixup) {
        return function (data, cc) {
          if (fixup instanceof Function) {
            data = fixup(data);
          }
          ajax({
            type:        'POST',
            url:         url,
            contentType: 'application/json; charset=utf-8',
            data:        JSON.stringify(data),
            success:     cc
          });
        };
      },

      updater: function (url, fixup) {
        return function (uuid, data, cc) {
          if (fixup instanceof Function) {
            data = fixup(data);
          }
          ajax({
            type:        'PUT',
            url:         url.replace(/%s/, uuid),
            contentType: 'application/json; charset=utf-8',
            data:        JSON.stringify(data),
            success:     cc
          });
        };
      },

      deleter: function (url) {
        var attr = 'uuid'; // might become a formal one day, who knows?
        return function (uuid, cc) {
          ajax({
            type: 'DELETE',
            url:  url.replace(/%s/, uuid),
            success: cc
          });
        };
      }

    };
  })();

  var Job = {};
  var Target = {};
  var Store = {};
  var Archive = {};
  var Schedule = {};
  var Retention = {};
  var Task = {};

  $.extend(Job, {
    type:   'job',
    list:   MODEL.lister  ('/v1/jobs', 'jobs'),
    delete: MODEL.deleter ('/v1/job/%s'),
    show:   MODEL.findOne ('/v1/job/%s'),
    create: MODEL.creater ('/v1/jobs'),
    update: MODEL.updater ('/v1/job/%s')
  });
  $.extend(Target, {
    type:   'target',
    list:   MODEL.lister  ('/v1/targets', 'targets'),
    delete: MODEL.deleter ('/v1/target/%s'),
    show:   MODEL.findOne ('/v1/target/%s'),
    create: MODEL.creater ('/v1/targets',   function (t) { t.plugin = t.plugin.toLowerCase();
                                                           t.endpoint = rejson(t.endpoint);
                                                           return t; }),
    update: MODEL.updater ('/v1/target/%s', function (t) { t.plugin = t.plugin.toLowerCase();
                                                           t.endpoint = rejson(t.endpoint);
                                                           return t; })
  });
  $.extend(Store, {
    type:   'store',
    list:   MODEL.lister  ('/v1/stores', 'stores'),
    delete: MODEL.deleter ('/v1/store/%s'),
    show:   MODEL.findOne ('/v1/store/%s'),
    create: MODEL.creater ('/v1/stores',   function (s) { s.plugin = s.plugin.toLowerCase();
                                                          s.endpoint = rejson(s.endpoint);
                                                          return s; }),
    update: MODEL.updater ('/v1/store/%s', function (s) { s.plugin = s.plugin.toLowerCase();
                                                          s.endpoint = rejson(s.endpoint);
                                                          return s; })
  });
  $.extend(Archive, {
    type:   'archive',
    list:   MODEL.lister   ('/v1/archives', 'archives'),
    search: MODEL.searcher ('/v1/archives', 'archives'),
    delete: MODEL.deleter  ('/v1/archive/%s'),
    show:   MODEL.findOne  ('/v1/archive/%s'),
    create: MODEL.creater  ('/v1/archives'),
    update: MODEL.updater  ('/v1/archive/%s')
  });
  $.extend(Schedule, {
    type:   'schedule',
    list:   MODEL.lister  ('/v1/schedules', 'schedules'),
    delete: MODEL.deleter ('/v1/schedule/%s'),
    show:   MODEL.findOne ('/v1/schedule/%s'),
    create: MODEL.creater ('/v1/schedules'),
    update: MODEL.updater ('/v1/schedule/%s')
  });
  $.extend(Retention, {
    type:   'retention-policy',
    list:   MODEL.lister  ('/v1/retention', 'retention'),
    delete: MODEL.deleter ('/v1/retention/%s'),
    show:   MODEL.findOne ('/v1/retention/%s', function (r) { r.expires = parseInt(r.expires / 86400); return r; }),
    create: MODEL.creater ('/v1/retention',    function (r) { r.expires = parseInt(r.expires) * 86400; return r; }),
    update: MODEL.updater ('/v1/retention/%s', function (r) { r.expires = parseInt(r.expires) * 86400; return r; })
  });
  $.extend(Task, {
    type: 'task',
    show: MODEL.findOne ('/v1/task/%s')
  });



  /* DatePicker

     Generate a DOM element containing a multi-month
     date picker widget that shows the current month
     and (w x h) preceding months, in a (w x h) grid.
   */
  function datepicker(w, h) {
    WDAYS = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];
    MONS  = ['January', 'February', 'March',
             'April',   'May',      'June',
             'July',    'August',   'Septermber',
             'October', 'November', 'December'];
    NUMS  = ['01', '02', '03', '04', '05', '06',
             '07', '08', '09', '10', '11', '12',
             '13', '14', '15', '16', '17', '18',
             '19', '20', '21', '22', '23', '24',
             '25', '26', '27', '28', '29', '30', '31'];

    today = new Date();
    var year = today.getFullYear();
    var month = today.getMonth();

    var M = [];
    /*
       work backwards through the grid in order to elegantly:
        - start with the current month and work backwards in time
        - determine if cells below / to the right of us are for different
          years (and therefore need a stronger border to differentiate)
     */
    for (y = h-1; y >= 0; y--) {
      M[y] = [];
      for (x = w-1; x >= 0; x--) {
        M[y][x] = {
          title:   MONS[month] + ' ' + year.toString(),
          prefix:  year.toString() + NUMS[month],
          primary: year,
          first:   (new Date(year, month,   1)).getDay(),  /* first weekday */
          last:    (new Date(year, month+1, 0)).getDate(), /* last day of month */
          css:     [(year % 2 == 0 ? 'even' : 'odd')]
        };
        /* is the cell to our right different from us? */
        if (x < w-1 && M[y][x+1].primary != M[y][x].primary) { M[y][x].css.push('right') }
        /* is the cell below us different from us? */
        if (y < h-1 && M[y+1][x].primary != M[y][x].primary) { M[y][x].css.push('bottom') }

        /* soooo last month... */
        month--; if (month < 0) { month = 11; year--; }
      }
    }

    var s = [];
    s.push('<div id="date-picker">');
    s.push('<table class="date year"><tbody><tr>');

    for (y = 0; y < h; y++) {
      s.push('<tr>');
      for (x = 0; x < w; x++) {
        var v = M[y][x];
        s.push('<td class="'+v.css.join(' ')+'"><table class="date month"><caption>'+v.title+'</caption><tr>');
        s.push('<th>' + WDAYS.join('</th><th>') + '</th>');
        s.push('</tr><tr>')

        for (j = 0; j < v.first + v.last; j++) {
          if (j % 7 == 0)  { s.push('</tr><tr>') }
          if (j < v.first) { s.push('<td></td>'); continue }
          s.push('<td data-value="'+v.prefix+NUMS[j - v.first]+'">'+(j - v.first + 1).toString()+'</td>');
        }
        s.push('</tr></table></td>');
      }
      s.push('</tr>');
    }

    s.push('</table>');
    s.push('</div>');
    return s.join('');
  }

  function select(list, id, display) {
    var s = '';
    for (var i = 0; i < list.length; i++) {
      s += '<option value="'+list[i][id]+'">'+list[i][display]+'</option>';
    }
    return s;
  }
  function timediff(a, offset) {
    var d = new Date();
    d.setTime(a.getTime() + offset);
    return d;
  }
  function yyyymmdd(d) {
    return (1900 + d.getYear()) * 10000 + d.getMonth() * 100 + d.getDay();
  }
  function weirdtime(s) {
    // because we insist on doing bad things with date formatting in the API...
    m = /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/.exec(s);
    if (!m) { return NaN }

    d = new Date();
    d.setTime(Date.UTC(parseInt(m[1]),   // year
                       parseInt(m[2])-1, // month
                       parseInt(m[3]),   // day
                       parseInt(m[4]),   // hour
                       parseInt(m[5]),   // minute
                       parseInt(m[6]))); // second
    return d;
  }
  function ago(d) {
    return duration(d, new Date()) + " ago";
  }
  function until(d) {
    return "in " + duration(d, new Date());
  }
  function duration(a_, b_) {
    if (isNaN(a_) || isNaN(b_)) {
      return "&infin;";
    }
    a = a_.getTime();
    b = b_.getTime();

    t = parseInt(Math.abs(a - b) / 1000);
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
  function dated(d) {
    if (isNaN(d)) { return "?"; }
    var wday = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    var mon  = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    var ord  = ['', 'st', 'nd', 'rd', 'th', 'th', 'th', 'th', 'th', 'th', 'th', //  1 - 10
                    'th', 'th', 'th', 'th', 'th', 'th', 'th', 'th', 'th', 'th', // 11 - 20
                    'st', 'nd', 'rd', 'th', 'th', 'th', 'th', 'th', 'th', 'th', // 21 - 30
                    'st'];
    var pre  = ['0', '0', '0', '0', '0', '0', '0', '0', '0', '0', //  0 -  9
                '',  '',  '',  '',  '',  '',  '',  '',  '',  '',  // 10 - 19
                '',  '',  '',  '',  '',  '',  '',  '',  '',  '',  // 20 - 29
                '',  '',  '',  '',  '',  '',  '',  '',  '',  '',  // 30 - 39
                '',  '',  '',  '',  '',  '',  '',  '',  '',  '',  // 40 - 49
                '',  '',  '',  '',  '',  '',  '',  '',  '',  '']; // 50 - 59
    md = d.getDate();
    hr = d.getHours() % 12; hr = hr == 0 ? 12 : hr;
    am = d.getHours() < 12;
    mn = d.getMinutes();
    return wday[d.getDay()]+" "+mon[d.getMonth()]+" "+md.toString()+ord[md]+
           " at "+hr.toString()+":"+pre[mn]+mn.toString()+(am?"am" : "pm");
  }
  function datecol(d, f) {
    return dated(d) + '<span class="sub">' + f(d) + '</span>';
  }

  function Validator(form) {
    this.form = $(form);
    this.data = this.form.serializeObject();
    this.seen = {}; /* keyed field-name => bool */
    this.okay = true;
  }
  Validator.prototype.ok = function () {
    return this.okay;
  };
  Validator.prototype.pass = function (name) {
    var c = this.form.find('[name="'+name+'"]').closest('.form-group');
    c.removeClass('has-warning has-error');
    c.find('.form-error').empty().hide();
    c.find('.help-block').show();
    return this;
  };
  Validator.prototype.fail = function (name, message) {
    this.okay = false;
    if (!(name in this.seen)) {
      this.seen[name] = true;

      var c = this.form.find('[name="'+name+'"]').closest('.form-group');
      c.removeClass('has-warning has-success').addClass('has-error');
      c.find('.help-block').hide();
      c.find('.form-error').empty().append(message).show();
    }
    return this;
  };
  Validator.prototype.present = function (name, message) {
    if (!(name in this.data) || this.data[name] == "") {
      return this.fail(name, message);
    }
    return this.pass(name);
  };
  Validator.prototype.integer = function (name, message) {
    if (!(name in this.data) || parseInt(this.data[name]) == NaN) {
      return this.fail(name, message);
    }
    return this.pass(name);
  };
  Validator.prototype.range = function (name, lo, hi, message) {
    if (!(name in this.data) || isNaN(parseInt(this.data[name]))) {
      return this.fail(name, message);
    }
    var n = parseInt(this.data[name]);
    if (n < lo || n > hi) {
      return this.fail(name, message);
    }
    return this.pass(name);
  };
  Validator.prototype.match = function (name, re, message) {
    if (!(name in this.data) || !re.test(this.data[name])) {
      return this.fail(name, message);
    }
    return this.pass(name);
  };
  Validator.prototype.json = function (name, message) {
    try {
      d = JSON.parse(this.data[name]);
      if ((d instanceof Array) || !(d instanceof Object)) { throw 'bad type' }
    } catch (e) {
      return this.fail(name, message);
    }
    return this.pass(name);
  };

  function innercell(o) {
    if (o instanceof Object) {
      if ("icon" in o) {
        return '<span class="glyphicon glyphicon-'+o.icon+' '+o.icon+'"></span>';
      }
    }
    if (typeof o == "undefined") {
      return "-";
    }
    return o.toString();
  }
  function cell(o) {
    if (o instanceof Array) {
      l = []
      for (var i = 0; i < o.length; i++) {
        l.push(innercell(o[i]));
      }
      return '<td class="icons icons-'+l.length.toString()+'">' + l.join("") + '</td>';
    }
    return '<td>' + innercell(o) + '</td>';
  }

  function cssify(s) {
    if (typeof s === 'undefined') { return '' }
    return s.toLowerCase().replace(/ /g, '-');
  }

  function summarize(text, max) {
    if (!max) {
      max = 80;
    }
    words = text.split(' ');
    safe = [];
    summary = words[0];
    while (words.length > 0 && summary.length < max) {
      w = words.shift();
      safe.push(w);
      summary += " " + words[0];
    }

    if (words.length == 0) {
      return safe.join(' ');
    }
    return safe.join(' ').replace(/[^a-zA-Z0-9]*$/, '...');
  }

  function Icon(name) {
    return {icon: name};
  }
  function Icons() {
    l = [];
    for (var i = 0; i < arguments.length; i++) {
      l.push({icon: arguments[i]});
    }
    return l;
  }

  var Loader = '<div id="tout"><div><div><div> </div></div></div></div>';
  function Loading(root) {
    root.append($(Loader));
  }

  function Header(title, button) {
    if (typeof button === 'undefined') {
      button = "";
    }
    if (button != "") {
      button = '<button class="btn btn-primary btn-md pull-right '+cssify(button)+'">'+button+'</button>';
    }
    return $('<h2 class="row"><div class="col-sm-8">'+title+'</div>'+
                             '<div class="col-sm-4">'+button+'</div></h2>');
  }

  function Table() {
    this.rows = [];
    this.type = arguments[0];
    this.headers = [].slice.call(arguments, 1)
    this.empty = "nothing to report";
  }
  Table.prototype.Empty = function(s) {
    this.empty = s;
    return this;
  };
  Table.prototype.Row = function() {
    this.rows.push([].slice.call(arguments))
    return this;
  };
  Table.prototype.Render = function(opts) {
    if (typeof opts === 'undefined') {
      opts = {};
    }
    if (this.rows.length == 0) {
      return '<div class="no-data">'+this.empty+'</div>';
    }
    var thead = "<thead><tr>";
    for (var i = 0; i < this.headers.length; i++) {
      thead += "<th>"+this.headers[i]+"</th>"
    }
    thead += "</tr></thead>";

    var tbody = "<tbody>";
    for (var i = 0; i < this.rows.length; i++) {
      css = opts.rowClass ? ' class="'+this.rows[i][0][opts.rowClass]+'"' : '';
      tbody += '<tr'+css+' data-uuid="'+this.rows[i][0].uuid+'">';
      for (var j = 1; j < this.rows[i].length; j++) {
        tbody += cell(this.rows[i][j]);
      }
      tbody += "</tr>";
    }
    tbody += "</tbody>";

    return $('<table class="table table-striped table-hover type-'+this.type+' '+this.type+'">'+thead+tbody+'</table>');
  };

  function Modal(html) {
    $('#modal').remove();
    if (html != '') {
      html = '<div class="main">'+html+'</div>';
    }
    $(document.body).append($('<div id="modal"><div class="fg">'+html+'</div></div>'));
  }
  $(document.body).on('click', '#modal', function (event) {
    if ($(event.target).is('#modal')) {
      $(event.target).remove();
    }
  });

  function Form(id, title, thing) {
    this.thing   = thing;
    this.id      = id;
    this.title   = title;
    this.buttons = [];
    this.fields  = [];
    this.Group();
  }
  function merge() {
    o = arguments[0];
    for (var i = 1; i < arguments.length; i++) {
      if (arguments[i]) {
        for (var key in arguments[i]) {
          o[key] = arguments[i][key];
        }
      }
    }
    return o;
  }
  Form.prototype.Field = function(name, label, opts) {
    opts = merge({
      type:    'text',  /* type of control, i.e. 'text', 'textarea', etc. */
      help:    '',      /* optional help text to be displayed below field. */
      listof:  '',      /* index into DB object for getting list; forces type:'select' */
      from:    '',      /* url to retrieve collection from, if DB[listof] does not exist. */
      keyed:   'uuid',  /* what field of the object to use for the value in a 'select' */
      auto:    false,   /* enable browser autocomplete; off by default */
      display: function(x) { return x.name }
    }, opts);
    if (opts.listof) {
      opts.type = 'select';
      if (opts.from == '') { opts.from = '/v1/'+opts.listof }
      if (!(opts.listof in DB)) {
        $.ajax({
          type: 'GET',
          url:  opts.from,
          async: false,
          success: function(l) { DB[opts.listof] = l; }
        });
      }
    }
    opts.name  = name;
    opts.label = label;
    this.fields[this.fields.length-1].fields.push(opts);
  };
  Form.prototype.Buttons = function() {
    for (var i = 0; i < arguments.length; i++) {
      this.buttons.push(arguments[i]);
    }
  };
  Form.prototype.Sidebar = function(s) {
    this.fields[this.fields.length-1].sidebar = s;
  };
  Form.prototype.Group = function() {
    this.fields.push({fields:[]});
  };
  Form.prototype.Render = function() {
    s = '';
    for (var i = 0; i < this.fields.length; i++) {
      var g = this.fields[i];
      s += '<div class="row"><div class="col-sm-6">';
      for (var j = 0; j < g.fields.length; j++) {
        var f = g.fields[j];
        var v = this.thing ? this.thing[f.name] : '';
        n='name="'+f.name+'" class="form-control"';
        s += '<div class="form-group"><label class="control-label">'+ f.label +'</label>';
        switch (f.type) {
        case 'select':
          s += '<select '+n+'>';
          if ('dummy' in f) {
            s += '<option value="">'+f.dummy+'</option>';
          }
          for (var i = 0; i < DB[f.listof].length; i++) {
            o = DB[f.listof][i];
            sel = '';
            if (this.thing && this.thing[f.name + '_' + f.keyed] == o[f.keyed]) {
              sel = ' selected="selected"';
            }
            s += '<option value="'+o[f.keyed]+'"'+sel+'>'+f.display(o)+'</option>';
          }
          s += '</select>';
          break;
        case 'textarea': s += '<textarea '+n+'>'+v+'</textarea>'; break;
        default:         s += '<input type="text" value="'+v+'" '+n+' autocomplete="'+(f.auto?"on":"off")+'"/>'; break;
        }
        if (f.help) { s += '<span class="help-block">'+f.help+'</span>'; }
        s += '<span class="form-error"></span>';
        s += '</div>';
      }
      s += '</div><div class="col-sm-6">';
      if (g.sidebar) {
        s += '<div class="bg-info form-sidebar">'+g.sidebar+'</div>';
      }
      s += '</div></div>';
    }
    s += '<div class="row"><div class="col-sm-6">';
    var l = []
    for (var i = 0; i < this.buttons.length; i++) {
      var b = this.buttons[i];
      if (b instanceof Object) {
        if ("ok" in b) {
          l.push('<button class="btn btn-primary">'+b.ok+'</button>');
        } else if ("cancel" in b) {
          l.push('<button class="btn cancel">'+b.cancel+'</button>');
        }
      } else {
        l.push('<button class="btn unknown">'+b+'</button>');
      }
    }
    s += l.join(' ');
    s += '</div>';
    return '<div id="'+this.id+'" class="container">'+
             '<h2>'+this.title+'</h2>'+
             '<form'+(this.thing ? ' data-uuid="'+this.thing.uuid+'"' : '')+'>'+s+'</form>'+
           '</div>';
  };

  var PAGES = {
    "#dashboard": function(root) {
      Loading(root);
      ajax({
        type: 'GET',
        url:  '/v1/tasks?active=t',
        success: function (tasks) {
          root.empty().append('<div class="row">'+
            '<div class="col-md-6" id="running"></div>'+
            '<div class="col-md-6" id="completed"></div>'+
            '</div>');

          tbl = new Table("task", "", "Type", "Owner", "Status", "Started at")
                   .Empty("No running tasks");
          for (var i = 0; i < tasks.length; i++) {
            t = tasks[i];
            start = weirdtime(t.started_at);
            tbl.Row(t, Icons("remove"), t.type, t.owner, t.status, datecol(start, ago));
          }
          root.find('#running').append(Header("Running Tasks", "")).append(tbl.Render());

          ajax({
            type: 'GET',
            url:  '/v1/tasks?active=f&limit=15',
            success: function (tasks) {
              tbl = new Table("task", "Type", "Owner", "Status", "Started at", "Duration")
                       .Empty("No completed tasks");
              for (var i = 0; i < tasks.length; i++) {
                t = tasks[i];
                start = weirdtime(t.started_at);
                end = weirdtime(t.stopped_at);
                tbl.Row(t, t.type, t.owner, t.status, datecol(start, ago), duration(start, end));
              }
              root.find('#completed').append(Header("Completed Tasks", "")).append(tbl.Render());
            }
          });
        }
      });
    },

    "#about": function (root) {
      Loading(root);
      ajax({
        type: 'GET',
        url:  '/v1/status',
        success: function (status) {
          root.empty().append(
              '<div id="about">'+
                '<h1>S.H.I.E.L.D.</h1>'+
                '<img src="../img/shield-med.png">'+
                '<pre class="system-health">'+
                  JSON.stringify(status, null, 4)+'</pre>'+
              '</div>');
        }
      });
    },

    "#jobs": function(root) {
      Loading(root);
      Job.list({ cached: false }, function () {
        tbl = new Table("job", "", "Name", "Target", "Store", "Schedule", "Retention", "");
        for (var i = 0; i < DB.jobs.length; i++) {
          j = DB.jobs[i];
          tbl.Row(j, Icons("repeat", j.paused ? "play" : "pause"), j.paused ? j.name + ' (paused)' : j.name, j.target_name, j.store_name, j.schedule_when + " ("+j.schedule_name+")", j.retention_name, Icons("edit", "trash"));
        }
        root.empty().append(Header("Jobs", "Create New Job")).append(tbl.Render());
      });
    },
    '#create-job': function(root) {
      form = new Form('create-job', 'New Backup Job');
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Job Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('target', 'Target System', {
        help: 'Where the data to be backed up resides.',
        listof: 'targets',
        dummy:  ''
      });
      form.Field('store', 'Storage Backend', {
        help:   'Where the backup archives should be stored.',
        listof: 'stores',
        dummy:  ''
      });
      form.Field('schedule', 'Schedule', {
        listof: 'schedules',
        dummy:  '',
        display: function(x) { return x.name + ' - ' + x.when }
      });
      form.Field('retention', 'Retention Policy', {
        listof: 'retention',
        dummy:  '',
        display: function(x) { return x.name + ' - keep for ' + (x.expires/86400).toString() + ' days'; }
      });
      form.Buttons({ok:'Create'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },
    '#edit-job': function(root, job) {
      form = new Form('update-job', 'Job: ' + job.name, job);
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Job Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('target', 'Target System', {
        help: 'Where the data to be backed up resides.',
        listof: 'targets',
        dummy:  ''
      });
      form.Field('store', 'Storage Backend', {
        help:   'Where the backup archives should be stored.',
        listof: 'stores',
        dummy:  ''
      });
      form.Field('schedule', 'Schedule', {
        listof: 'schedules',
        dummy:  '',
        display: function(x) { return x.name + ' - ' + x.when }
      });
      form.Field('retention', 'Retention Policy', {
        listof: 'retention',
        dummy:  '',
        display: function(x) { return x.name + ' - keep for ' + (x.expires/86400).toString() + ' days'; }
      });
      form.Buttons({ok:'Update'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },

    '#archives': function(root) {
      Loading(root);
      Target.list(function () {
        root.empty().append($(
          '<h2>Backups</h2>'+
          '<div class="container">'+
            '<form id="archive-search" class="form-inline">'+
                '<div class="form-group">'+
                  ' <label for="archiveStart">From</label> '+
                  '<input type="text" id="archiveStart" name="start" class="form-control date" placeholder="'+yyyymmdd(timediff(new Date(), 86400 * 30))+'"/>'+
                '</div> '+
                '<div class="form-group">'+
                  '<label for="archiveEnd">Until</label> '+
                  '<input type="text" id="archiveEnd" name="end" class="form-control date" placeholder="'+yyyymmdd(new Date())+'"/>'+
                '</div> '+
                '<div class="form-group">'+
                  '<label for="archiveTarget">Target System</label> '+
                  '<select id="archiveTarget" name="archiveTarget" class="form-control">'+
                    select(DB.targets, 'uuid', 'name')+
                  '</select>'+
                '</div> '+
                '<div class="form-group">'+
                  '<button>go</button> '+
                '</div>'+
            '</form>'+
          '</div>'+
          '<div class="container main" id="archives-main"></div>'
        ));
        if (LOCAL.archive_search_form_submitted) {
          $('#main #archive-search').trigger('submit');
        }
      });
    },

    '#stores': function(root) {
      Loading(root);
      Store.list({ cached: false }, function () {
        tbl = new Table("store", "Name", "Summary", "Plugin", "Configuration", "");
        for (var i = 0; i < DB.stores.length; i++) {
          s = DB.stores[i];
          tbl.Row(s, s.name, summarize(s.summary), s.plugin, "<pre>"+JSON.stringify(JSON.parse(s.endpoint),null,2)+"</pre>", Icons("edit", "trash"));
        }
        root.empty().append(Header("Stores", "Create New Store")).append(tbl.Render());
      });
    },
    '#create-store': function(root) {
      form = new Form('create-store', 'New Storage Backend');
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Store Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('plugin', 'Plugin Name', { help: 'The name of the backup plugin to use when sending data to this system' });
      form.Field('endpoint', 'Configuration (JSON)', { type: 'textarea', help: 'A JSON object of properties' });
      form.Buttons({ok:'Create'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },
    '#edit-store': function(root, store) {
      form = new Form('update-store', 'Storage Backend: ' + store.name, store);
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Store Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('plugin', 'Plugin Name', { help: 'The name of the backup plugin to use when sending data to this system' });
      form.Field('endpoint', 'Configuration (JSON)', { type: 'textarea', help: 'A JSON object of properties' });
      form.Buttons({ok:'Update'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },

    '#targets': function(root) {
      Loading(root);
      Target.list({ cached: false }, function () {
        tbl = new Table("target", "Name", "Summary", "Plugin", "Configuration", "");
        for (var i = 0; i < DB.targets.length; i++) {
          t = DB.targets[i];
          tbl.Row(t, t.name, summarize(t.summary), t.plugin, "<pre>"+JSON.stringify(JSON.parse(t.endpoint),null,2)+"</pre>", Icons("edit", "trash"));
        }
        root.empty().append(Header("Targets", "Create New Target")).append(tbl.Render());
      });
    },
    '#create-target': function(root) {
      form = new Form('create-target', 'New Target System');
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Target Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('plugin', 'Plugin Name', { help: 'The name of the backup plugin to use when retrieving data from this system' });
      form.Field('endpoint', 'Configuration (JSON)', { type: 'textarea', help: 'A JSON object of properties to use during backup' });
      form.Field('agent', 'Remote IP:port', { help: 'The IP address and port of the remote system' });
      form.Buttons({ok:'Create'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },
    '#edit-target': function(root, target) {
      form = new Form('update-target', 'Target System: ' + target.name, target);
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Target Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('plugin', 'Plugin Name', { help: 'The name of the backup plugin to use when retrieving data from this system' });
      form.Field('endpoint', 'Configuration (JSON)', { type: 'textarea', help: 'A JSON object of properties to use during backup' });
      form.Field('agent', 'Remote IP:port', { help: 'The IP address and port of the remote system' });
      form.Buttons({ok:'Update'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },

    '#schedules': function(root) {
      Loading(root);
      Schedule.list({ cached: false }, function () {
        tbl = new Table("schedule", "Name", "Summary", "When", "");
        for (var i = 0; i < DB.schedules.length; i++) {
          s = DB.schedules[i];
          tbl.Row(s, s.name, summarize(s.summary), s.when, Icons("edit", "trash"));
        }
        root.empty().append(Header("Schedules", "Create New Schedule")).append(tbl.Render());
      });
    },
    '#create-schedule': function(root) {
      form = new Form('create-schedule', 'New Schedule');
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Schedule Name', { help: 'Try to give your backup schedule a unique and memorable name' });
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('when', 'Schedule');
      form.Buttons({ok:'Create'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },
    '#edit-schedule': function(root, schedule) {
      form = new Form('update-schedule', 'Schedule: ' + schedule.name, schedule);
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Schedule Name', { help: 'Try to give your backup schedule a unique and memorable name' });
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('when', 'Schedule');
      form.Buttons({ok:'Update'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },

    '#retention': function(root) {
      Loading(root);
      Retention.list({ cached: false }, function () {
        tbl = new Table("retention", "Name", "Summary", "Expiration", "");
        for (var i = 0; i < DB.retention.length; i++) {
          p = DB.retention[i];
          tbl.Row(p, p.name, summarize(p.summary), p.expires / 86400 + " days", Icons("edit", "trash"));
        }
        root.empty().append(Header("Retention Policies", "Create New Retention Policy")).append(tbl.Render());
      });
    },
    '#create-retention-policy': function(root) {
      form = new Form('create-retention-policy', 'New Retention Policy');
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Policy Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('expires', 'Expiration (in days)', { help: 'How many days to keep backup archives' });
      form.Buttons({ok:'Create'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    },
    '#edit-retention-policy': function(root, policy) {
      form = new Form('update-retention-policy', 'Retention Policy: ' + policy.name, policy);
      //form.Sidebar('lorem ipsum dolor sit amet');
      form.Field('name', 'Policy Name');
      form.Field('summary', 'Summary', { type: 'textarea' });
      form.Field('expires', 'Expiration (in days)', { help: 'How many days to keep backup archives' });
      form.Buttons({ok:'Update'}, {cancel:'Cancel'});
      root.empty().append(form.Render()).find('form .form-control:first').focus();
    }
  };

  $(function() {
    var go = function() {
      var to = arguments[0];
      var args = [].slice.call(arguments, 1);
      if (!(to instanceof Array)) { to = [to] }

      for (var i = 0; i < to.length; i++) {
        if (PAGES[to[i]]) {
          if ($('.navbar-nav li a[href="'+to[i]+'"]').length > 0) {
            $('.navbar-nav li').removeClass('active');
            $('.navbar-nav li a[href="'+to[i]+'"]').closest('li').addClass('active');
          }
          args.unshift($('#main').empty());
          $('#notif').empty();
          PAGES[to[i]].apply(null, args);
          return true;
        }
      }
      return false;
    };
    $(window).on('hashchange', function() {
      go(document.location.hash);
    });

    var createForm = lastly(function(event) {
      go('#create-'+event.data.model.type);
    });
    $('#main').on('click', 'button.create-new-retention-policy', { model: Retention }, createForm);
    $('#main').on('click', 'button.create-new-schedule',         { model: Schedule  }, createForm);
    $('#main').on('click', 'button.create-new-target',           { model: Target    }, createForm);
    $('#main').on('click', 'button.create-new-store',            { model: Store     }, createForm);
    $('#main').on('click', 'button.create-new-job',              { model: Job       }, createForm);

    var validateForm = function(event) {
      var validate = new Validator($(event.target).closest('form'));

      switch (event.data.model.type) {
      case 'retention-policy':
        validate.present('name',    "Please provide a name for this retention policy")
                .integer('expires', "Expiration must be specified as a number (of days)")
                .range(  'expires', 1, 3660,
                                    "Please specify an expiration period between 1 day and 10 years");
        break;

      case 'schedule':
        validate.present('name', "Please provide a name for this backup schedule")
                .present('when', "Required.  Example: 'daily at 4am' or 'every week at 2:30am on monday'");
        break;

      case 'target':
        validate.present('name',     "Please provide a name for this target system")
                .present('plugin',   "Please select the plugin used to back up this system")
                .present('endpoint', "Required.  Consult plugin documentation for format")
                .json(   'endpoint', "Configuration must be a well-formed JSON object")
                .match(  'agent',    /^.+:\d+$/,
                                     "Not an IP:port or hostname:port...");
        break;

      case 'store':
        validate.present('name',     "Please provide a name for this storage backend")
                .present('plugin',   "Please select the plugin used to access this store")
                .present('endpoint', "Required.  Consult plugin documentation for format")
                .json(   'endpoint', "Configuration must be a well-formed JSON object");
        break;

      case 'job':
        validate.present('name',      "Please provide a name for this backup job")
                .present('target',    "Select a target system to backup")
                .present('store',     "Select where to store the backups")
                .present('schedule',  "When should this backup job run?")
                .present('retention', "What policy governs retention of backup archives?");
        break;
      }

      return validate.ok();
    };
    $('#main').on('blur', '#create-retention-policy form.attempted [name]', { model: Retention }, validateForm);
    $('#main').on('blur', '#update-retention-policy form.attempted [name]', { model: Retention }, validateForm);
    $('#main').on('blur', '#create-schedule form.attempted [name]',         { model: Schedule  }, validateForm);
    $('#main').on('blur', '#update-schedule form.attempted [name]',         { model: Schedule  }, validateForm);
    $('#main').on('blur', '#create-target form.attempted [name]',           { model: Target    }, validateForm);
    $('#main').on('blur', '#update-target form.attempted [name]',           { model: Target    }, validateForm);
    $('#main').on('blur', '#create-store form.attempted [name]',            { model: Store     }, validateForm);
    $('#main').on('blur', '#update-store form.attempted [name]',            { model: Store     }, validateForm);
    $('#main').on('blur', '#create-job form.attempted [name]',              { model: Job       }, validateForm);
    $('#main').on('blur', '#update-job form.attempted [name]',              { model: Job       }, validateForm);

    var createThing = lastly(function(event) {
      if (validateForm(event) != true) { return }
      event.data.model.create(
          $(event.target).closest('form').addClass('attempted').serializeObject(),
          function () { go(event.data.next) });
    });
    $('#main').on('submit', '#create-retention-policy form', { model: Retention, next: '#retention' }, createThing);
    $('#main').on('submit', '#create-schedule form',         { model: Schedule,  next: '#schedules' }, createThing);
    $('#main').on('submit', '#create-target form',           { model: Target,    next: '#targets'   }, createThing);
    $('#main').on('submit', '#create-store form',            { model: Store,     next: '#stores'    }, createThing);
    $('#main').on('submit', '#create-job form',              { model: Job,       next: '#jobs'      }, createThing);

    var deleteThing = lastly(function(event) {
      if (!confirm("Are you sure you want to delete this?\n\n(Once deleted, it cannot be un-deleted)")) { return }
      Loading($('#main'));

      event.data.model.delete(
          $(event.target).closest('[data-uuid]').data('uuid'),
          function () { go(event.data.next) });
    });
    $('#main').on('click', '.retention [data-uuid] .trash', { model: Retention, next: '#retention' }, deleteThing);
    $('#main').on('click', '.schedule  [data-uuid] .trash', { model: Schedule,  next: '#schedules' }, deleteThing);
    $('#main').on('click', '.archive   [data-uuid] .trash', { model: Archive,   next: '#archives'  }, deleteThing);
    $('#main').on('click', '.target    [data-uuid] .trash', { model: Target,    next: '#targets'   }, deleteThing);
    $('#main').on('click', '.store     [data-uuid] .trash', { model: Store,     next: '#stores'    }, deleteThing);
    $('#main').on('click', '.job       [data-uuid] .trash', { model: Job,       next: '#jobs'      }, deleteThing);

    var editThing = lastly(function(event) {
      Loading($('#main'));
      event.data.model.show(
          $(event.target).closest('[data-uuid]').data('uuid'),
          function (thing) { go(event.data.form, thing) });
    });
    $('#main').on('click', '.retention [data-uuid] .edit', { model: Retention, form: '#edit-retention-policy' }, editThing);
    $('#main').on('click', '.schedule [data-uuid]  .edit', { model: Schedule,  form: '#edit-schedule'         }, editThing);
    $('#main').on('click', '.target [data-uuid]    .edit', { model: Target,    form: '#edit-target'           }, editThing);
    $('#main').on('click', '.store [data-uuid]     .edit', { model: Store,     form: '#edit-store'            }, editThing);
    $('#main').on('click', '.job [data-uuid]       .edit', { model: Job,       form: '#edit-job'              }, editThing);

    var updateThing = lastly(function(event) {
      if (validateForm(event) != true) { return }

      Loading($('#main'));
      var $form = $(event.target).closest('form');
      event.data.model.update(
          $form.data('uuid'),
          $form.serializeObject(),
          function () { go(event.data.next) });
    });
    $('#main').on('submit', '#update-retention-policy form', { model: Retention, next: '#retention' }, updateThing);
    $('#main').on('submit', '#update-schedule form',         { model: Schedule,  next: '#schedules' }, updateThing);
    $('#main').on('submit', '#update-target form',           { model: Target,    next: '#targets'   }, updateThing);
    $('#main').on('submit', '#update-store form',            { model: Store,     next: '#stores'    }, updateThing);
    $('#main').on('submit', '#update-job form',              { model: Job,       next: '#jobs'      }, updateThing);

    var pauseJob = lastly(function(event) {
      Loading($('#main'));
      uuid = $(event.target).closest('[data-uuid]').data('uuid');
      ajax({
        type: 'POST',
        url:  '/v1/job/'+uuid+(event.data.pause ? '/pause' : '/unpause'),
        success: function() {
          go('#jobs');
          if (event.data.pause) {
            notify('success', 'Paused Job &mdash; SHIELD will not schedule this job to run automatically.');
          } else {
            notify('success', 'Unpaused Job &mdash; SHIELD will start scheduling this job to run automatically.');
          }
        }
      });
    });
    $('#main').on('click', '.job [data-uuid] .pause', { pause: true  }, pauseJob);
    $('#main').on('click', '.job [data-uuid] .play',  { pause: false }, pauseJob);

    var runJob = lastly(function(event) {
      Loading($('#main'));
      uuid = $(event.target).closest('[data-uuid]').data('uuid');
      ajax({
        type: 'POST',
        url:  '/v1/job/'+uuid+'/run',
        success: function() {
          go('#jobs');
          notify('success', 'Scheduled an immediate run of the backup job');
        }
      });
    });
    $('#main').on('click', '.job [data-uuid] .repeat', runJob);

    var searchArchives = lastly(function(event) {
      LOCAL.archive_search_form_submitted = true;
      Loading($('#archives-main'));
      Archive.search({
          after:  $('#archiveStart').val(),
          before: $('#archiveEnd').val(),
          target: $('#archiveTarget').val(),
          limit:  75
        }, function (_, archives) {
          DB.archives = archives;
          Target.list(function () {
            Store.list(function () {
              var T = indexby('targets', 'uuid');
              var S = indexby('stores',  'uuid');

              var tbl = new Table("archive", "", "Target", "Restore IP", "Store", "Taken at", "Expires at", "Status", "Notes", "");
              for (var i = 0; i < DB.archives.length; i++) {
                var a = DB.archives[i];
                var t = T[a.target_uuid];
                var s = S[a.store_uuid];
                tbl.Row(a, Icons("import"), t.name, t.agent, s.name, a.taken_at, a.expires_at, a.status, a.notes, Icons("trash"));
              }
              $('#archives-main').empty().append(tbl.Render({rowClass: 'status'}));
            });
          })
        });
    });
    $('#main').on('submit', '#archive-search', searchArchives);

    var restoreArchive = lastly(function(event) {
      Loading($('#main'));
      uuid = $(event.target).closest('[data-uuid]').data('uuid');
      ajax({
        type: 'POST',
        url:  '/v1/archive/'+uuid+'/restore'
      });
    });
    $('#main').on('click', '.archive [data-uuid] .import', restoreArchive);

    $('#main').on('click', '.task tr[data-uuid] .remove', lastly(function (event) {
      uuid = $(event.target).closest('[data-uuid]').data('uuid');
      ajax({
        type: 'DELETE',
        url:  '/v1/task/'+uuid,
        success: function() {
          go('#dashboard');
        }
      });
    }));
    $('#main').on('click', '.task tr[data-uuid]', function (event) {
      Modal('');
      $('#modal .fg').append($(Loader));
      Task.show($(event.target).closest('[data-uuid]').data('uuid'),
      function (task) {
        t = $.extend({}, task);
        delete t.uuid;
        delete t.type;
        delete t.status;
        delete t.log;

        var start = weirdtime(task.started_at);
        var end   = weirdtime(task.stopped_at);

        var header = [
          [['Owner', task.owner],   ['Started',  dated(start)]],
          [['Status', task.status], ['Finished', dated(end)]],
          [['',      ''],           ['Duration', duration(start, end)]],
          [['',      ''],           ['', '']]
        ];

        n = 2;
        if (task.job_uuid)     { header[n++][0] = ['Job', task.job_uuid]; }
        if (task.archive_uuid) { header[n++][0] = ['Archive', task.archive_uuid]; }

        table = '<table>';
        for (var y = 0; y < header.length; y++) {
          table += '<tr>';
          for (var x = 0; x < header[y].length; x++) {
            table += '<th>'+header[y][x][0]+'</th><td>'+header[y][x][1]+'</td>';
          }
          table += '</tr>';
        }
        table += '</table>';


        Modal(
          '<div>'+
            '<h2 class="'+task.status+'">'+task.type+' task '+task.uuid+'</h2>'+
            (task.owner == "system"
             ? '<p class="autorun">this task was run automatically, by SHIELD, as part of its normal operation</p>'
             : '')+
            '<header>' + table + '</header>'+
            '<h3>Task Log</h3>'+
            '<pre class="terminal">'+(task.log == "" ? '(no output from task)' : task.log)+'</pre>'+
          '</div>'
        );
      });
    });


    $(document.body).on('click', '#notif div', lastly(function (event) {
      $(event.target).fadeOut();
    }));
    $('#main').on('click', 'button.unknown', lastly(noop));
    $('#main').on('click', 'button.cancel', lastly(function(event) {
      go(document.location.hash);
    }));
    $('#main').on('click', 'input.date', lastly(function (event) {
      $(this).addClass('date-target');
      $(document.body).append(datepicker(6, 3));
    }));
    $(document.body).on('click', 'table.date td[data-value]', lastly(function (event) {
      var value = $(event.target).data('value');
      $('input.date-target').val(value).removeClass('date-target');
      $('#date-picker').remove();
    }));
    $(document.body).on('click', '#date-picker', lastly(function (event) {
      var what = $(event.target);
      if (!what.is('#date-picker')) { return }
      what.remove();
    }));

    go([document.location.hash, '#dashboard']);
  });

})(window, document)
// vim:et:sts=2:ts=2:sw=2
