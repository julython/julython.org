var JULY = JULY || {};

JULY.ViewModel = function(options) {
	this.initialize.apply(this,arguments);
};
_.extend(JULY.ViewModel.prototype,{
	
	initialize: function() {}

});
JULY.ViewModel.extend=Backbone.View.extend;

JULY.applyBindings = function(e,t) {
	var n=$(t);
	n.length>0?ko.applyBindings(e,n[0]):console.log('Binding error:  no elements found for "'+t+'"')
};

JULY.Commit = Backbone.Model.extend({
	// Commit model
	url: '/api/v1/commit/'
});

JULY.PlayerCommits = Backbone.Collection.extend({
	model: JULY.Commit,
	
	url: function() {return '/api/v1/commit/?' + this.params()},
	
	initialize: function(data, options) {
		this.user = options.userId;
		this.limit = options.limit || 20;
		this.offset = options.offset || 0;
	},
	
	params: function() {return jQuery.param({
		user: this.userId,
		limit: this.limit,
		offset: this.offset,
	})}
	
});

JULY.ProjectCommits = Backbone.Collection.extend({
	model: JULY.Commit,
	
	url: function() {return '/api/v1/commit/?' + this.params()},
	
	initialize: function(data, options) {
		this.projectId = options.projectId;
		this.limit = options.limit || 20;
		this.offset = options.offset || 0;
		this.total = 0;
	},
	
	params: function() {return jQuery.param({
		project: this.projectId,
		limit: this.limit,
		offset: this.offset,
	})},
	
	parse: function(resp) {
		this.total = resp.meta.total_count;
		this.offset = resp.meta.offset + this.limit;
		return resp.objects;
	}
	
});

JULY.ProjectCommitsView = JULY.ViewModel.extend({
	
	initialize: function(options) {
		this.c = new JULY.ProjectCommits(null, options);
		this.c.fetch({add: true});
		this.commits = kb.collectionObservable(this.c);
	},
	
	fetch: function(){
		this.commits.collection().fetch({add:true});
	}
});

