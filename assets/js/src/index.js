/* VARIOUS HELPERS for HOMEPAGE */
if (!$) { var $ = null; }
if (!console) { var console = null; }

var Nav = function(selector, offset) {
  this._selector = selector;
  this._el = $(selector);
  if ($(window).width() < 979) {
  	this._offset = 0;
  } else {
	this._offset = (offset || 0);
  }
  this._startTop = this._el.position().top;
  this._sectionPadding = 20; // arbitrary?
  this._startPadding = Number(this._el.next().css("padding-top").replace("px", ""));
  this._height = this._el.outerHeight();
};

Nav.prototype.setup = function() {
  var self = this;
  this._el.find("button").each(function() {
    var buttonEl = $(this);
    buttonEl.click(function() {
      var scrollToEl = $("h2."+this.className);
      var scrollTo = scrollToEl.position().top - self._height - self._offset - self._sectionPadding;
      $("html, body").animate({scrollTop: scrollTo}, 1100);
    });
  });
};

Nav.prototype.update = function() {
  var win = $(window);
  if (win.scrollTop() + this._offset < this._startTop) {
    this._el.removeClass("fixed");
    this._el.next().css("padding-top", this._startPadding+"px");
    this._el.css("top", "");
    this._el.css("width", "");
  } else {
    this._el.addClass("fixed");
    this._el.css("top", this._offset+"px");
    this._el.next().css("padding-top", (this._startPadding + this._height)+"px");
    this._el.css("width", "100%");
  }
};


