var JULY = JULY || {};

JULY.Badge = Backbone.Model.extend({
  url: function() {
    return '/api/v1/user/' + this.userId + '/badge/' + this.params();
  }
});


JULY.BadgeView = function(model) {
  // Bind the model to the view
  kb.viewModel(model, {}, this);

  // Calculated Elements
  this.status = this.awarded() ?
    "Awarded (" + this.total() + ")" :
    this.count() + " / " + this.total();
  this.progress = this.awarded() ? "100%" :
    Math.round((this.count() / this.total()) * 100) + "%";
  this.badge_classes = this.level() + " " + (this.awarded() ? "awarded": "unawarded");

  // Overlay observables
  var self = this;
  self.isVisible = ko.observable();
  self.topPosition = ko.observable();
  self.leftPosition = ko.observable();

  this.toggleVisible = function(element) {
    var el = $(element);
    var position = el.position();
    // Adjust the position of the overlay
    self.topPosition(position.top + el.height() + 15 + "px");
    self.leftPosition(position.left + "px");
    // Now toggle the visiblity
    self.isVisible(!self.isVisible());
  };
};


JULY.BadgesCollection = Backbone.Collection.extend({

  model: JULY.Badge,

  url: function() {return '/api/v1/user/'+ this.userId + '/badges/'},

  initialize: function(data, options) {
    this.userId = options.userId;
  },

  parse: function(resp) {
    return resp.badges;
  }
});

JULY.BadgesView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.badgesCollection = new JULY.BadgesCollection(null, options);
    this.badgesCollection.fetch();
    this.badges = kb.collectionObservable(this.badgesCollection, {view_model: JULY.BadgeView});
  }

});
