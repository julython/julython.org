var JULY = JULY || {};

ko.bindingHandlers.pageBottom={
	init:function(e,t,n,r){
		var i=t(),s=n(),o=s.callbackThreshold||1e3,u=s.callbackInterval||500;
		if(typeof i!="function")throw new Error("The value of the pageBottom binding must be a function");
		i=$.proxy(i,r);
		var a=$(document),f=$(window);
		setInterval(function(){a.height()-f.height()-f.scrollTop()<o&&i()},u)
	}
};

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

JULY.CommitCollection = Backbone.Collection.extend({
	model: JULY.Commit,
	
	url: function() {return '/api/v1/commit/?' + this.params()},
	
	initialize: function(data, options) {
		this.projectId = options.projectId;
		this.userId = options.userId;
		this.limit = options.limit || 20;
		this.offset = options.offset || 0;
		this.total = 0;
	},
	
	params: function() {
		var p = {limit: this.limit, offset: this.offset}
		if (this.projectId) {p.project = this.projectId}
		if (this.userId) {p.user = this.userId}
		return jQuery.param(p);
	},
	
	parse: function(resp) {
		this.total = resp.meta.total_count;
		this.offset = resp.meta.offset + this.limit;
		return resp.objects;
	}
	
});

JULY.CommitsView = JULY.ViewModel.extend({
	
	initialize: function(options) {
		this.c = new JULY.CommitCollection(null, options);
		this.c.fetch({add: true});
		this.commits = kb.collectionObservable(this.c);
		this.hasMore = ko.computed(function() {
			return this.c.length < this.c.total}, this);
	},
	
	scrolled: function(data, event) {
        var elem = event.target;
        if (elem.scrollTop > (elem.scrollHeight - elem.offsetHeight - 200)) {
            this.fetch();
        }
    },
	
	fetch: function(){
		this.commits.collection().fetch({add:true});
	}
});

