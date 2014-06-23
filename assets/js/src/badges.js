var JULY = JULY || {};

JULY.Badge = Backbone.Model.extend({
  url: function() {
    return '/api/v1/user/' + this.userId + '/badge/' + this.params();
  }
});


JULY.BadgeView = function(model) {
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
  this.badge_classes = this.level + " " + this.awarded_class;
  this.status_classes = (this.awarded) ?
    "fa-check-square" :
    "fa-refresh";
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
