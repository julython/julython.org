var JULY = JULY || {};

JULY.Group = Backbone.Model.extend({
  // generic group
  url: '/api/v1/'
});

JULY.GroupCollection = Backbone.Collection.extend({
  model: JULY.Group,

  initialize: function(data, options) {
  	var options = options || {}
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
      p['name__istartswith'] = this.query;
    }
    return jQuery.param(p);
  },

  filter: function(query) {
  	this.total = 0;
  	this.offset = 0;
  	this.query = query;
  	this.fetch();
  	return this.map(function(m) { return m.get('name')});
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
