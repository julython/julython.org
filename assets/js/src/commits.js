var JULY = JULY || {};

JULY.Commit = Backbone.Model.extend({
  // Commit model
  url: '/api/v1/commit/'
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
      this.commits.collection().fetch({add:true});
    }
  }
});

