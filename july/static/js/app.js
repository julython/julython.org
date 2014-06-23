var JULY = JULY || {};

/*
 * Custom bindings
 */
ko.bindingHandlers.pageBottom={
  init:function(e,t,n,r){
    var i=t(),s=n(),o=s.callbackThreshold||1e3,u=s.callbackInterval||500;
    if(typeof i!="function")throw new Error("The value of the pageBottom binding must be a function");
    i = $.proxy(i,r);
    var a=$(document),f=$(window);
    setInterval(function () {
      if (a.height() - f.height() - f.scrollTop() < o) {
        i();
      }
    },u);
  }
};

ko.bindingHandlers.timeago = {
    init: function(element, valueAccessor, allBindingsAccessor) {
        // First get the latest data that we're bound to
        var value = valueAccessor();
        allBindingsAccessor();

        // Next, whether or not the supplied model property is observable,
        // get its current value
        var valueUnwrapped = ko.utils.unwrapObservable(value);

        // set the title attribute to the value passed
        $(element).attr('title', valueUnwrapped);

        // apply timeago to change the text of the element
        $(element).timeago();
    },
    update: function(element, valueAccessor, allBindingsAccessor) {
        var value = valueAccessor();
        allBindingsAccessor();
        var valueUnwrapped = ko.utils.unwrapObservable(value);
        $(element).timeago('update', valueUnwrapped);
    }
};

ko.bindingHandlers.typeahead = {
	init: function(element, valueAccessor, allBindingsAccessor) {
		var value = valueAccessor();
		allBindingsAccessor();
		// unwrap the source function
		var source = ko.utils.unwrapObservable(value);
		// Apply typeahead to the item
		$(element).typeahead({
		  source: source
		});
	}
};

/*
 *  Base ViewModel
 * 
 *  This object mimics the backbone style extend model to apply 
 *  attributes to the prototype.
 * 
 */
JULY.ViewModel = function(options) {
  this.initialize.apply(this,arguments);
};
_.extend(JULY.ViewModel.prototype,{
  initialize: function() {}
});
JULY.ViewModel.extend=Backbone.View.extend;

/*
 * Custom bind function
 */
JULY.applyBindings = function(e,t) {
  var n=$(t);
  if (n.length > 0) {
    ko.applyBindings(e,n[0]);
  } else {
    console.log('Binding error:  no elements found for "'+t+'"');
  }
};

JULY.csrfSafeMethod = function(method) {
    // these HTTP methods do not require CSRF protection
    return (/^(GET|HEAD|OPTIONS|TRACE)$/.test(method));
};

JULY.setCSRFToken = function(csrftoken) {
  jQuery.ajaxSetup({
    crossDomain: false, // obviates need for sameOrigin test
    beforeSend: function(xhr, settings) {
      if (!JULY.csrfSafeMethod(settings.type)) {
        xhr.setRequestHeader("X-CSRFToken", csrftoken);
      }
    }
  });
};

$(document).ready(function(){
    var $send_abuse = $('#send-abuse');
    var $form = $('#abuse-form');
    var $modal = $('#abuse-modal');
    var $abuseli = $('#abuseli');
    $send_abuse.click(function(){
        var desc = $form.find('textarea').val();
        if(!desc) return false;
        $.ajax({
            type: $form.attr('method'),
            url: $form.attr('action'),
            data: $form.serialize()
        });
        $modal.find('.modal-footer').html('');
        $modal.find('.modal-body').html('<h4>Thank you !</h4>');
        $send_abuse.remove();
        $abuseli.remove();
        return false;
    })
});

var JULY = JULY || {};

JULY.Badge = Backbone.Model.extend({
  url: function() {
    return '/api/v1/user/' + this.userId + '/badge/' + this.params();
  }
});


JULY.BadgeView = function(model) {
  this.badge_id = model.get("badge_id");
  this.title = model.get("title");
  this.subtitle = model.get("subtitle");
  this.count = model.get("count");
  this.total = model.get("total");
  this.icon = model.get("icon");
  this.level = model.get("level");
  this.awarded = model.get("awarded");
  this.status = (this.awarded) ?
    "Awarded (" + this.total + ")" :
    this.count + " / " + this.total;
  this.awarded_class = (this.awarded) ?
    "awarded" :
    "unawarded";
  this.progress = Math.round((this.count / this.total) * 100);
  if (this.progress > 100) { this.progress = 100; }
  this.progress = this.progress + '%';
  this.badge_classes = this.level + " " + this.awarded_class;
  this.status_classes = (this.awarded) ?
    "fa-check-square" :
    "fa-refresh";
  this.over = function(b) {
    var el = $("[data-badge-id='" + b.badge_id + "']");
    var position = el.position();
    var left = position.left;
    var top = position.top + el.height() + 15;
    var overlay = $("[data-overlay-for='" + b.badge_id + "']");
    overlay.css({top: top + "px", left: left + "px"});
    overlay.show();
  };
  this.out = function(b) {
    $("[data-overlay-for='" + b.badge_id + "']").hide();
  };
};


JULY.BadgesCollection = Backbone.Collection.extend({

  model: JULY.Badge,

  url: function() {return '/api/v1/user/'+ this.userId + '/badges/?' + this.params();},

  initialize: function(data, options) {
    this.userId = options.userId;
    this.hasMore = false;
  },

  params: function() {
    return jQuery.param({});
  },

  parse: function(resp) {
    var badges = [];
    resp.badges.forEach(function(badge, index) {
      badge.badge_id = index;
      badge.count = resp.total_commits;
      badges.push(badge);
    });
    return badges;
  }
});

JULY.BadgesView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.badgesCollection = new JULY.BadgesCollection(null, options);
    this.badgesCollection.fetch();
    this.badges = kb.collectionObservable(this.badgesCollection, {view_model: JULY.BadgeView});
  },

  fetch: function(){
    this.badges.collection().fetch();
  }

});

var JULY = JULY || {};

JULY.Project = Backbone.Model.extend({
  // Project model
  url: '/api/v1/project/'
});


JULY.BoardCollection = Backbone.Collection.extend({
  model: JULY.Project,

  url: function() {return '/api/v1/largeboard/?' + this.params();},

  initialize: function(data, options) {
    this.limit = options.limit || 20;
    this.offset = options.offset || 0;
    this.total = 0;
    this.hasMore = false;
  },

  params: function() {
    var p = {
      limit: this.limit,
      offset: this.offset
    };
    return jQuery.param(p);
  },

  parse: function(resp) {
    this.total = resp.meta.total_count;
    this.offset = resp.meta.offset + this.limit;
    this.hasMore = this.total > this.models.length;
    return resp.objects;
  }

});


JULY.LargeBoardCollection = JULY.BoardCollection.extend({

  url: function() {return '/api/v1/largeboard/?' + this.params();}

});

JULY.MediumBoardCollection = JULY.BoardCollection.extend({

  url: function() {return '/api/v1/mediumboard/?' + this.params();}

});

JULY.SmallBoardCollection = JULY.BoardCollection.extend({

  url: function() {return '/api/v1/smallboard/?' + this.params();}

});

JULY.BoardView = function(model){
  this.points = kb.observable(model, 'points');
  this.project = model.get('project');
  this.url = "/projects/" + this.project.slug + "/";
};

JULY.BoardsView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.l = new JULY.LargeBoardCollection(null, options);
    this.m = new JULY.MediumBoardCollection(null, options);
    this.s = new JULY.SmallBoardCollection(null, options);
    this.l.fetch({add: true});
    this.m.fetch({add: true});
    this.s.fetch({add: true});
    this.largeBoards = kb.collectionObservable(this.l, {view_model: JULY.BoardView});
    this.mediumBoards = kb.collectionObservable(this.m, {view_model: JULY.BoardView});
    this.smallBoards = kb.collectionObservable(this.s, {view_model: JULY.BoardView});
  },

  largeHasMore: function() {
    return this.largeBoards.collection().hasMore;
  },

  mediumHasMore: function() {
    return this.mediumBoards.collection().hasMore;
  },

  smallHasMore: function() {
    return this.smallBoards.collection().hasMore;
  },

  scrolled: function(data, event) {
        var elem = event.target;
        if (elem.scrollTop > (elem.scrollHeight - elem.offsetHeight - 200)) {
            this.fetch();
        }
    },

  fetch: function(){
    if (this.largeHasMore()) {
      this.largeBoards.collection().fetch({remove:false});
    }
    if (this.mediumHasMore()) {
      this.mediumBoards.collection().fetch({remove:false});
    }
    if (this.smallHasMore()) {
      this.smallBoards.collection().fetch({remove:false});
    }
  }
});
(function() {

  var BarChart = function(chartId, dataUrl) {
    this._chartId = chartId;
    this._dataUrl = dataUrl;
    this._chartElement = $("#"+chartId);
    this._width = this._chartElement.width();
    this._height = this._chartElement.height();
    this._pad = {
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      barX: 1,
      barY: 1
    };
    this._options = {};
    // sane defaults -- overwrite on a per-element basis with CSS.
    if (!this._width || !this._height) {
      this._width = 600;
      this._height = 400;
    }
    this._chart = d3.select("#"+this._chartId).append("svg");
    this._chart
      .attr("class", "chart")
      .attr("width", this._width)
      .attr("height", this._height);
  };

  BarChart.prototype.load = function(stats) {
    this._stats = stats;
    return this;
  };

  BarChart.prototype.set = function(option, value) {
    var func = this["_set_"+option];
    if (!func) {
      console.log("No set for option '"+option+"'.");
    } else {
      func.apply(this, [value]);
    }
    return this;
  };

  BarChart.prototype._set_xLabel = function(xLabel) {
    this._options.xLabel = xLabel;
    this._pad.bottom += 24;
  };

  BarChart.prototype._set_title = function(title) {
  };

  BarChart.prototype._set_yLabel = function(yLabel) {
    this._pad.top += 20;
    this._options.yLabel = yLabel;
  };

  BarChart.prototype._getChartDimensions = function() {
    var numBars = this._stats.length;
    var availableWidth = (this._width - (this._pad.left + this._pad.right)) -
      (numBars * this._pad.barX);
    var availableHeight = this._height - (this._pad.bottom + this._pad.top);
    var barWidth = availableWidth / numBars;
    var maxValue = 1;
    for (var i=0; i<numBars; i++) {
      if (this._stats[i] > maxValue) {
        maxValue = this._stats[i];
      }
    }
    return {
      bars: numBars,
      barWidth: barWidth,
      maxValue: maxValue,
      width: availableWidth,
      height: availableHeight,
      top: this._pad.top,
      bottom: this._height - this._pad.bottom,
      left: this._pad.left,
      right: this._width - this._pad.right
    };
  };

  BarChart.prototype.render = function() {
    var size = this._getChartDimensions();
    var rect = this._chart.selectAll("rect");
    var data = rect.data(this._stats);
    var enter = data.enter();
    var self = this;
    this._addFilters();
    enter.append("rect")
      .attr("x", function(d, n) {
        return self._pad.left + n * (size.barWidth + 1);
      })
      .attr("y", function(d, n) {
        return size.top + (size.height - _getHeight(d, size.maxValue, size.height));
      })
      .attr("width", size.barWidth)
      .attr("height", function(d, n) {
        return _getHeight(d, size.maxValue, size.height);
      });
    for (var key in this._options) {
      if (this._options.hasOwnProperty(key)) {
        var func = this["_render_"+key];
        if (!func) {
          console.log("No rendering option for '"+key+"'.");
        } else {
          func.apply(this, [enter, this._options[key]]);
        }
      }
    }
    return this;
  };

  BarChart.prototype._addFilters = function() {
    var filter = this._chart.append("filter")
      .attr("id", "dropshadow");
    filter.append("feGaussianBlur")
      .attr("in", "SourceAlpha")
      .attr("stdDeviation", 0.1);

    filter.append("feOffset")
      .attr("dx", 0)
      .attr("dy", 1)
      .attr("result", "offsetblur");

    var merge = filter.append("feMerge");
    merge.append("feMergeNode");
    merge.append("feMergeNode")
      .attr("in", "SourceGraphic");
  };

  BarChart.prototype._render_xLabel = function(entered, labelValue) {
    var size = this._getChartDimensions();
    var self = this;
    entered.append("text")
      .attr("class", "xLabel")
      .attr("x", function(d, i) {
        return self._pad.left +
          (size.barWidth + self._pad.barX) * i +
          (size.barWidth / 2);
      })
      .attr("y", function(d, i) {
        return self._pad.top + size.height + self._pad.bottom - 3;
      })
      .attr("width", size.barWidth)
      .attr("text-anchor", "middle")
      .text(function(d, i) {
        return _getValue(labelValue, d, i);
      });
    this._chart.append("line")
      .attr("class", "rule")
      .attr("y1", size.height + this._pad.top + 2)
      .attr("y2", size.height + this._pad.top + 2)
      .attr("x1", this._pad.left - 2)
      .attr("x2", this._pad.left + size.width + (size.bars * this._pad.barX));
  };

  BarChart.prototype._render_yLabel = function(entered, labelValue) {
    var size = this._getChartDimensions();
    var self = this;
    entered.append("text")
      .attr("class", "yLabel")
      .attr("text-anchor", "middle")
      .attr("width", size.barWidth)
      .attr("x", function(d, i) {
        return self._pad.left +
          (size.barWidth + self._pad.barX) * i +
          (size.barWidth / 2);
      })
      .attr("y", function(d, i) {
        console.log(size);
        return self._pad.top + (
          size.height - _getHeight(d, size.maxValue, size.height)) - 5;
      })
      .text(function(d, i) {
        return _getValue(labelValue, d, i);
      });
  };

  var _getHeight = function(value, maxValue, maxHeight) {
    var height = (value / maxValue) * maxHeight;
    return height;
  };

  var _getValue = function(getValue, point, index) {
    if (!$.isFunction(getValue)) {
      return getValue;
    }
    return getValue(point, index);
  };

  window.Charts = {
    BarChart: BarChart
  };

})();

var JULY = JULY || {};

JULY.Commit = Backbone.Model.extend({
  // Commit model
  url: '/api/v1/commit/'
});

JULY.CommitCalendarDay = Backbone.Model.extend({
  idAttribute: 'timestamp',
  date: function() {
    var d = new Date(this.get('timestamp'));
    return d3.time.day(d);
  }
});

JULY.CommitCalendar = Backbone.Collection.extend({
  model: JULY.CommitCalendarDay,

  url: function(){
    return '/api/v1/commit/calendar/?' + this.params();
  },

  initialize: function(data, options){
    this.username = options.username || null;
    this.start = null;
    this.end = null;
  },

  start_date: function() {
    return d3.time.day(this.start);
  },

  parse: function(resp) {
    this.start = new Date(resp.start);
    this.end = new Date(resp.end);
    var s = d3.time.day(this.start);
    var e = d3.time.day(this.end);
    // start and end are off by one
    var start = d3.time.day.offset(s, 1);
    var end = d3.time.day.offset(e, 1);
    var allDays = d3.time.days(start, end);
    var formatter = d3.time.format('%Y-%m-%d');

    // Use it to create an empty commit calendar.
    var data = _(allDays).map(function(day){
      var calendarDay = {
        'timestamp': formatter(day),
        'commit_count': 0
      };
      return calendarDay;
    });
    this.reset(data);
    return resp.objects;
  },

  params: function(){
    var p = {};
    if (this.username){
      p.username = this.username;
    }
    return jQuery.param(p);
  }
});


JULY.CommitCollection = Backbone.Collection.extend({
  model: JULY.Commit,

  url: function() {return '/api/v1/commit/?' + this.params();},

  initialize: function(data, options) {
    this.projectId = options.projectId;
    this.userId = options.userId;
    this.languages = options.languages || [];
    this.limit = options.limit || 20;
    this.offset = options.offset || 0;
    this.total = 0;
    this.hasMore = false;
    this._pushStream = new PushStream({
      host: window.location.hostname,
      port: window.location.port,
      modes: "websocket|stream",
      urlPrefixStream: "/events/sub",
      urlPrefixPublisher: "/events/pub",
      urlPrefixWebsocket: "/events/ws"
    });
    JULY.collection = this;
    this._pushStream.onmessage = function(text) {
      console.log("-- New commit from PushStream:", text);
      JULY.collection.unshift(text);
    };
    this._pushStream.onstatuschange = function(state) {
      console.log("-- PushStream state changed: " + state);
      if (state === PushStream.CLOSED) {
        console.log("!! ERROR: PushStream was closed.");
      }
    };
    var channel = (this.projectId) ? "project-" + this.projectId : (this.userId) ? "user-" + this.userId: "global";
    console.log("-- Subscribing to PushStream channel: " + channel);
    this._pushStream.addChannel(channel);
    this._pushStream.connect();
  },

  params: function() {
    var p = {
      limit: this.limit,
      offset: this.offset
    };
    if (this.projectId) {
      p.project = this.projectId;
    }
    if (this.userId) {
      p.user = this.userId;
    }
    if (this.languages) {
      p.languages = _(this.languages).reduce(function(memo, language, index) {
        var notFirst = index > 0;
        return memo.concat((notFirst?';':'') + language);
      }, '', this);
    }
    return jQuery.param(p);
  },

  parse: function(resp) {
    this.total = resp.meta.total_count;
    this.offset = resp.meta.offset + this.limit;
    this.hasMore = this.total > this.models.length;
    return resp.objects;
  }

});

JULY.CommitsView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.c = new JULY.CommitCollection(null, options);
    this.c.fetch({add: true});
    this.commits = kb.collectionObservable(this.c);
  },

  hasMore: function() {
    return this.commits.collection().hasMore;
  },

  scrolled: function(data, event) {
        var elem = event.target;
        if (elem.scrollTop > (elem.scrollHeight - elem.offsetHeight - 200)) {
            this.fetch();
        }
    },

  fetch: function(){
    if (this.hasMore()) {
      this.commits.collection().fetch({remove:false});
    }
  }
});

JULY.makeCalendar = function(elmentId, username) {
  // Calendar example: http://bl.ocks.org/mbostock/4063318
  // d3.chart quickstart: https://github.com/misoproject/d3.chart/wiki/quickstart

  // Fetch the user's actual commit calendar and use it to fill in correct
  // commit counts.
  var calendar = new JULY.CommitCalendar(null, {username: username});
  calendar.fetch({async:false, remove:false});

  // An array containing only the commit counts, to build the graphic of the calendar.
  var counts = calendar.map(function(day){return day.get('commit_count');});

  // Calendar dimensions.
  var cellSize = 12,
    day = d3.time.format("%w"),
    week = d3.time.format("%U"),
    width = cellSize * 7,
    height = cellSize * 6;

  // The color scale.
  var color = d3.scale.linear()
    .domain([0, Math.max.apply(null, counts)])
    .range(['white', 'darkgreen']);

  // The svg element.
  var svg = d3.select(elmentId)
    .attr('width', width)
    .attr('height', height);

  // Building the actual cells of the calendar.
  var cells = svg.selectAll('rect')
    .data(calendar.models)
    .enter().append('rect')
      .attr('width', cellSize).attr('height', cellSize)
      .attr('y', function(d) {
        return cellSize * ( week(d.date()) - week(calendar.start_date()));
      })
      .attr('x', function(d) {return cellSize * day(d.date());})
      .style('stroke', '#BEC9AF')
      .style('fill', function(d){
        return color(d.get('commit_count'));
      });
  cells.append("title")
    .text(function(d) {
      return d.get('commit_count') + ' commits on ' + d.get('timestamp'); });
};

var JULY = JULY || {};


JULY.parse_url = function(url){
  return url.replace('https://api.github.com/', '/api/v1/github/');
};


JULY.Repository = Backbone.Model.extend({

  initialize: function(data, options) {
    this.url = JULY.parse_url(data.url);
    this.hooks_url = JULY.parse_url(data.hooks_url);
    this.hooks = new JULY.HookCollection(null, {url: this.hooks_url});
    this.hooks.fetch();
  }

});

JULY.Hook = Backbone.Model.extend({
  initialize: function(data, options) {
    this.url = JULY.parse_url(data.url);
  }
});


JULY.RepositoryCollection = Backbone.Collection.extend({
  model: JULY.Repository,

  url: function() {return '/api/v1/github/user/repos?' + this.params();},

  initialize: function(data, options) {
    this.per_page = options.per_page || 20;
    this.page = options.page || 1;
    this.type = options.type || "public";
    this.total = 0;
    this.hasMore = true;
  },

  params: function() {
    var p = {
      per_page: this.per_page,
      page: this.page,
      type: this.type
    };
    return jQuery.param(p);
  },

  parse: function(resp) {
    this.total += resp.length;
    this.page += 1;
    this.hasMore = resp.length == this.per_page;
    return resp;
  }

});


JULY.HookCollection = Backbone.Collection.extend({
  model: JULY.Hook,

  url: function() {
    return this._url;
  },

  initialize: function(data, options) {
    this._url = options.url;
    this.per_page = options.per_page || 100;
    this.page = options.page || 1;
    this.total = 0;
    this.hasMore = false;
  },

  params: function() {
    var p = {
      per_page: this.per_page,
      page: this.page
    };
    return jQuery.param(p);
  },

  parse: function(resp) {
    var objs = [];
    _.each(resp, function(hook) {
      var config = hook.config || {};
      // TODO: regex?
      if (config.url == "http://www.julython.org/api/v1/github"){
        objs.push(hook);
      }
    });
    this.total += objs.length;
    this.page += 1;
    this.hasMore = objs.length == this.per_page;
    return objs;
  }

});

JULY.HookView = function(model){
  var self = this;
  this.last_response = model.get('last_response');
  this.test_url = JULY.parse_url(model.get('test_url'));
  this.updated_at = ko.observable(model.get('updated_at'));
  // Test the hook!
  this.test = function(csrftoken) {
    JULY.setCSRFToken(csrftoken);
    jQuery.post(self.test_url, {action: "test"},
      function(data){
        var d=new Date();
        self.updated_at(d.toISOString());
    });
  };
};

JULY.RepoView = function(model){
  var self = this;
  this.working = ko.observable(false);
  this.name = kb.observable(model, 'name');
  this.html_url = kb.observable(model, 'html_url');
  this.description = kb.observable(model, 'description');
  this.hooks = kb.collectionObservable(model.hooks, {view_model: JULY.HookView});
  // add the hook!
  this.add = function(csrftoken) {
    self.working(true);
    JULY.setCSRFToken(csrftoken);
    var url = self.hooks.collection().url;
    jQuery.post(url, {action: "add"},
      function(data){
        self.hooks.collection().reset([]);
        self.hooks.collection().fetch();
        self.working(false);
    });
  };
};

JULY.RepositoryView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.r = new JULY.RepositoryCollection(null, options);
    this.r.fetch();
    this.repos = kb.collectionObservable(this.r, {view_model: JULY.RepoView});
  },

  hasMore: function() {
    return this.repos.collection().hasMore;
  },

  scrolled: function(data, event) {
        var elem = event.target;
        if (elem.scrollTop > (elem.scrollHeight - elem.offsetHeight - 200)) {
            this.fetch();
        }
    },

  fetch: function(){
    if (this.hasMore()) {
      this.repos.collection().fetch({remove:false});
    }
  }
});

/* VARIOUS HELPERS for HOMEPAGE */
if (!$) { var $ = null; }
if (!console) { var console = null; }

var Nav = function(selector, offset) {
  this._selector = selector;
  this._el = $(selector);
  if ($(window).width() < 979) {
  	this._offset = 0;
  } else {
	this._offset = (offset || 0);
  }
  this._startTop = this._el.position().top;
  this._sectionPadding = 20; // arbitrary?
  this._startPadding = Number(this._el.next().css("padding-top").replace("px", ""));
  this._height = this._el.outerHeight();
};

Nav.prototype.setup = function() {
  var self = this;
  this._el.find("button").each(function() {
    var buttonEl = $(this);
    buttonEl.click(function() {
      var scrollToEl = $("h2."+this.className);
      var scrollTo = scrollToEl.position().top - self._height - self._offset - self._sectionPadding;
      $("html, body").animate({scrollTop: scrollTo}, 1100);
    });
  });
};

Nav.prototype.update = function() {
  var win = $(window);
  if (win.scrollTop() + this._offset < this._startTop) {
    this._el.removeClass("fixed");
    this._el.next().css("padding-top", this._startPadding+"px");
    this._el.css("top", "");
    this._el.css("width", "");
  } else {
    this._el.addClass("fixed");
    this._el.css("top", this._offset+"px");
    this._el.next().css("padding-top", (this._startPadding + this._height)+"px");
    this._el.css("width", "100%");
  }
};



var JULY = JULY || {};

JULY.Group = Backbone.Model.extend({
  // generic group
  url: '/api/v1/'
});

JULY.GroupCollection = Backbone.Collection.extend({
  model: JULY.Group,

  initialize: function(data, options) {
  	options = options || {};
    this.limit = options.limit || 20;
    this.offset = options.offset || 0;
    this.total = 0;
    this.query = '';
    this.hasMore = false;
  },

  params: function() {
    var p = {
      limit: this.limit,
      offset: this.offset
    };
    if (this.query) {
      p['name__icontains'] = this.query;
    }
    return jQuery.param(p);
  },

  filter: function(query) {
  	this.total = 0;
  	this.offset = 0;
  	this.query = query;
  	this.fetch();
  	return this.map(function(m) { return m.get('name');});
  },
  
  parse: function(resp) {
    this.total = resp.meta.total_count;
    this.offset = resp.meta.offset + this.limit;
    this.hasMore = this.total > this.models.length;
    return resp.objects;
  }
});

JULY.LocationCollection = JULY.GroupCollection.extend({
  url: function() {return '/api/v1/location/?' + this.params();}
});

JULY.TeamCollection = JULY.GroupCollection.extend({
  url: function() {return '/api/v1/team/?' + this.params();}
});

JULY.ProfileView = JULY.ViewModel.extend({
  initialize: function(options) {
    this.locations = kb.collectionObservable(new JULY.LocationCollection());
    this.teams = kb.collectionObservable(new JULY.TeamCollection());
    JULY.profile = this;
  },

  filterLocation: function(query) {
    return JULY.profile.locations.collection().filter(query);
  },

  filterTeam: function(query) {
    return JULY.profile.teams.collection().filter(query);
  }
});

JULY.UserProjectCollection = Backbone.Collection.extend({
  model: JULY.Project,

  url: function() {return '/api/v1/user/'+ this.userId + '/projects/?' + this.params();},

  initialize: function(data, options) {
    this.userId = options.userId;
    this.limit = options.limit || 20;
    this.offset = options.offset || 0;
    this.total = 0;
    this.hasMore = false;
  },

  params: function() {
    var p = {
      limit: this.limit,
      offset: this.offset
    };
    return jQuery.param(p);
  },

  parse: function(resp) {
    this.total = resp.meta.total_count;
    this.offset = resp.meta.offset + this.limit;
    this.hasMore = this.total > this.models.length;
    return resp.objects;
  }
});

JULY.ProjectsView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.p = new JULY.UserProjectCollection(null, options);
    this.p.fetch({add: true});
    this.projects = kb.collectionObservable(this.p);
  },

  hasMore: function() {
    return this.projects.collection().hasMore;
  },

  scrolled: function(data, event) {
        var elem = event.target;
        if (elem.scrollTop > (elem.scrollHeight - elem.offsetHeight - 200)) {
            this.fetch();
        }
    },

  fetch: function(){
    if (this.hasMore()) {
      this.projects.collection().fetch({remove:false});
    }
  }
});
