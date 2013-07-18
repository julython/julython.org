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
    console.log(this._url);
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
  console.log(model);
  var self = this;
  this.last_response = model.get('last_response');
  this.test_url = JULY.parse_url(model.get('test_url'));
  this.updated_at = ko.observable(model.get('updated_at'));
  // Test the hook!
  this.test = function(csrftoken) {
    JULY.setCSRFToken(csrftoken);
    jQuery.post(self.test_url, "",
      function(data){
        var d=new Date();
        self.updated_at(d.toISOString());
    });
  };
};

JULY.RepoView = function(model){
  console.log(model);
  var self = this;
  this.working = ko.observable(false);
  this.name = kb.observable(model, 'name');
  this.html_url = kb.observable(model, 'html_url');
  this.hooks = kb.collectionObservable(model.hooks, {view_model: JULY.HookView});
  // add the hook!
  this.add = function(csrftoken) {
    JULY.setCSRFToken(csrftoken);
    var POST = {
      "url": self.hooks.collection().url,
      "name": "web",
      "active": true,
      "events": ["push"],
      "config": {
        "url": "http://www.julython.org/api/v1/github",
        "content_type": "form",
        "insecure_ssl": "1"
      }
    };
    console.log(self.hooks.collection().url);
    self.hooks.collection().create(POST, {wait: true});
  };
};

JULY.RepositoryView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.r = new JULY.RepositoryCollection(null, options);
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
