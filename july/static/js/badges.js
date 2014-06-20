var Badge = function(data) {
  this.id = data.id;
  this.title = data.title;
  this.subtitle = data.subtitle;
  this.count = data.count;
  this.total = data.total;
  this.icon = data.icon;
  this.level = data.level;
  this.status_icon = (data.awarded) ?
    "<span class='fa fa-check-square'/> " :
    "<span class='fa fa-spin fa-spinner'/> ";
  this.status = (data.awarded) ? "Awarded (" + this.total + ")" :  this.count + " / " + this.total;
  this.awarded_class = (data.awarded) ? "awarded" : "unawarded";
  this.progress = Math.round((this.count / this.total) * 100);
};

Badge.prototype.render = function() {
  var self = this;
  var el = build($("[data-template='badge']"), self);
  el.on("mouseover", function() {
    var position = el.position();
    var left = position.left;
    var top = position.top + el.height() + 15;
    self.overlay.css({top: top + "px", left: left + "px"});
    self.overlay.show();
  });
  el.on("mouseout", function() { self.overlay.hide(); });

  self.overlay = build($("[data-template='badge-overlay']"), self);
  self.overlay.hide();
  $("body").append(self.overlay);

  return el;
};


var build = function(template, data) {
  var html = template.html();
  for (var attr in data) {
    if (!data.hasOwnProperty(attr)) {
      continue;
    }
    html = html.replace(new RegExp('{ ' + attr + ' }', 'g'), data[attr]);
  }
  return $(html);
};


window.onload = function() {
  var badgeContainer = $("[data-container='badges']");

  // temporary until we're returning from the API
  var badges = [
    {
      id: 1,
      title: "Committed",
      subtitle: "100 Commits",
      count: 100,
      total: 100,
      awarded: true,
      icon: "fa-trophy",
      level: "novice"
    },
    {
      id: 2,
      title: "Crazy",
      subtitle: "1,000 Commits",
      count: 426,
      total: 1000,
      awarded: false,
      icon: "fa-trophy",
      level: "journeyman"
    },
    {
      id: 3,
      title: "Insane",
      subtitle: "10,000 Commits",
      count: 426,
      total: 10000,
      awarded: false,
      icon: "fa-trophy",
      level: "expert"
    }
  ];

  for (var i = 0; i < badges.length; i++) {
    var badgeData = badges[i];
    var badge = new Badge(badgeData);
    var el = badge.render();
    badgeContainer.append(el);
  }

};
