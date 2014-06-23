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


JULY.Badge = Backbone.Model.extend({
  url: function() {
    return '/api/v1/user/' + this.userId + '/badge/' + this.params();
  }
});


JULY.BadgeView = function(model) {
  console.log(model);
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
  console.log(this.progress);
  this.badge_classes = this.level + " " + this.awarded_class;
  this.status_classes = (this.awarded) ?
    "fa-check-square" :
    "fa-spin fa-spinner";
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
