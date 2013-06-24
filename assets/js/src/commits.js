var JULY = JULY || {};

JULY.Commit = Backbone.Model.extend({
  // Commit model
  url: '/api/v1/commit/'
});

JULY.CommitCalendarDay = Backbone.Model.extend({
  idAttribute: 'timestamp'
});

JULY.CommitCalendar = Backbone.Collection.extend({
  model: JULY.CommitCalendarDay,

  url: function(){
    return '/api/v1/commit/calendar/?' + this.params();
  },

  initialize: function(data, options){
    this.username = options.username || null;
    this.days = options.days || null;
    this.end_date = options.date || null;
  },

  params: function(){
    var p = {};
    if (this.username){
      p.username = this.username;
    }
    if(this.days) {
      p.days = this.days;
    }
    if(this.end_date && this.end_date instanceof Date){
      p.end_date = this.end_date.toISOString();
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

  // Create an array of dates, 35 days back.
  var now = new Date();
  var today = d3.time.day(now);
  var then = d3.time.day.offset(now, -36);
  var allDays = d3.time.days(then, today);
  var formatter = d3.time.format('%Y-%m-%d');

  // Use it to create an empty commit calendar.
  var data = _(allDays).map(function(day){
    var calendarDay = {
      'timestamp': formatter(day),
      'commit_count': 0
    };
    return calendarDay;
  });

  // Fetch the user's actual commit calendar and use it to fill in correct
  // commit counts.
  var calendar = new JULY.CommitCalendar(data, {username: username});
  calendar.fetch({async:false, remove:false});

  // An array containing only the commit counts, to build the graphic of the calendar.
  data = calendar.map(function(day){return day.get('commit_count');});

  // Calendar dimensions.
  var cellSize = 10,
    border = 1,
    weekLength = 7,
    height = (cellSize + border * 2) * weekLength,
    width = (cellSize + border * 2) * 5;

  // The color scale.
  var color = d3.scale.linear()
    .domain([0, Math.max.apply(null, data)])
    .range(['white', 'green']);

  // The svg element.
  var svg = d3.select(elmentId)
    .attr('width', width)
    .attr('height', height)
    .style('background-color', '#BEC9AF');

  // Building the actual cells of the calendar.
  var cells = svg.selectAll('rect')
    .data(data)
    .enter().append('rect')
      .attr('width', cellSize).attr('height', cellSize)
      .attr('x', function(d,i) {
        var week = Math.floor(i / weekLength);
        return (cellSize + 2 * border) * week + border;
      })
      .attr('y', function(d,i) {
        var weekday = i % weekLength;
        return (cellSize + 2 * border) * weekday + border;
      })
      .style('fill', color);
};
