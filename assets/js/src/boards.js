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