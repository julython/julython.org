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
    // start and end are off by one
    this.start = new Date(resp.start);
    this.end = new Date(resp.end);
    var start = d3.time.day(this.start);
    var end = d3.time.day(this.end);
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
