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
    this.hasMore = false;
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

  initialize: function(data, options) {
    this.url = options.url;
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


JULY.RepoView = function(model){
  // TODO: fill me in
};

JULY.RepositoryView = JULY.ViewModel.extend({

  initialize: function(options) {
    //TODO: fill me in
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
