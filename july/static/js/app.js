var JULY = JULY || {};

/*
 * Custom bindings
 */
ko.bindingHandlers.pageBottom={
  init:function(e,t,n,r){
    var i=t(),s=n(),o=s.callbackThreshold||1e3,u=s.callbackInterval||500;
    if(typeof i!="function")throw new Error("The value of the pageBottom binding must be a function");
    i = $.proxy(i,r);
    var a=$(document),f=$(window);
    setInterval(function () {
      if (a.height() - f.height() - f.scrollTop() < o) {
        i();
      }
    },u);
  }
};

ko.bindingHandlers.timeago = {
    init: function(element, valueAccessor, allBindingsAccessor) {
        // First get the latest data that we're bound to
        var value = valueAccessor();
        allBindingsAccessor();

        // Next, whether or not the supplied model property is observable,
        // get its current value
        var valueUnwrapped = ko.utils.unwrapObservable(value);

        // set the title attribute to the value passed
        $(element).attr('title', valueUnwrapped);

        // apply timeago to change the text of the element
        $(element).timeago();
    }
};

ko.bindingHandlers.typeahead = {
	init: function(element, valueAccessor, allBindingsAccessor) {
		var value = valueAccessor();
		allBindingsAccessor();
		// unwrap the source function
		var source = ko.utils.unwrapObservable(value);
		// Apply typeahead to the item
		$(element).typeahead({
		  source: source
		});
	}
};

/*
 *  Base ViewModel
 * 
 *  This object mimics the backbone style extend model to apply 
 *  attributes to the prototype.
 * 
 */
JULY.ViewModel = function(options) {
  this.initialize.apply(this,arguments);
};
_.extend(JULY.ViewModel.prototype,{
  initialize: function() {}
});
JULY.ViewModel.extend=Backbone.View.extend;

/*
 * Custom bind function
 */
JULY.applyBindings = function(e,t) {
  var n=$(t);
  if (n.length > 0) {
    ko.applyBindings(e,n[0]);
  } else {
    console.log('Binding error:  no elements found for "'+t+'"');
  }
};

(function() {

  var BarChart = function(chartId, dataUrl) {
    this._chartId = chartId;
    this._dataUrl = dataUrl;
    this._chartElement = $("#"+chartId);
    this._width = this._chartElement.width();
    this._height = this._chartElement.height();
    this._pad = {
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      barX: 1,
      barY: 1
    };
    this._options = {};
    // sane defaults -- overwrite on a per-element basis with CSS.
    if (!this._width || !this._height) {
      this._width = 600;
      this._height = 400;
    }
    this._chart = d3.select("#"+this._chartId).append("svg");
    this._chart
      .attr("class", "chart")
      .attr("width", this._width)
      .attr("height", this._height);
  };

  BarChart.prototype.load = function(stats) {
    this._stats = stats;
    return this;
  };

  BarChart.prototype.set = function(option, value) {
    var func = this["_set_"+option];
    if (!func) {
      console.log("No set for option '"+option+"'.");
    } else {
      func.apply(this, [value]);
    }
    return this;
  };

  BarChart.prototype._set_xLabel = function(xLabel) {
    this._options.xLabel = xLabel;
    this._pad.bottom += 24;
  };

  BarChart.prototype._set_title = function(title) {
  };

  BarChart.prototype._set_yLabel = function(yLabel) {
    this._pad.top += 20;
    this._options.yLabel = yLabel;
  };

  BarChart.prototype._getChartDimensions = function() {
    var numBars = this._stats.length;
    var availableWidth = (this._width - (this._pad.left + this._pad.right)) -
      (numBars * this._pad.barX);
    var availableHeight = this._height - (this._pad.bottom + this._pad.top);
    var barWidth = availableWidth / numBars;
    var maxValue = 1;
    for (var i=0; i<numBars; i++) {
      if (this._stats[i] > maxValue) {
        maxValue = this._stats[i];
      }
    }
    return {
      bars: numBars,
      barWidth: barWidth,
      maxValue: maxValue,
      width: availableWidth,
      height: availableHeight,
      top: this._pad.top,
      bottom: this._height - this._pad.bottom,
      left: this._pad.left,
      right: this._width - this._pad.right
    };
  };

  BarChart.prototype.render = function() {
    var size = this._getChartDimensions();
    var rect = this._chart.selectAll("rect");
    var data = rect.data(this._stats);
    var enter = data.enter();
    var self = this;
    this._addFilters();
    enter.append("rect")
      .attr("x", function(d, n) {
        return self._pad.left + n * (size.barWidth + 1);
      })
      .attr("y", function(d, n) {
        return size.top + (size.height - _getHeight(d, size.maxValue, size.height));
      })
      .attr("width", size.barWidth)
      .attr("height", function(d, n) {
        return _getHeight(d, size.maxValue, size.height);
      });
    for (var key in this._options) {
      if (this._options.hasOwnProperty(key)) {
        var func = this["_render_"+key];
        if (!func) {
          console.log("No rendering option for '"+key+"'.");
        } else {
          func.apply(this, [enter, this._options[key]]);
        }
      }
    }
    return this;
  };

  BarChart.prototype._addFilters = function() {
    var filter = this._chart.append("filter")
      .attr("id", "dropshadow");
    filter.append("feGaussianBlur")
      .attr("in", "SourceAlpha")
      .attr("stdDeviation", 0.1);

    filter.append("feOffset")
      .attr("dx", 0)
      .attr("dy", 1)
      .attr("result", "offsetblur");

    var merge = filter.append("feMerge");
    merge.append("feMergeNode");
    merge.append("feMergeNode")
      .attr("in", "SourceGraphic");
  };

  BarChart.prototype._render_xLabel = function(entered, labelValue) {
    var size = this._getChartDimensions();
    var self = this;
    entered.append("text")
      .attr("class", "xLabel")
      .attr("x", function(d, i) {
        return self._pad.left +
          (size.barWidth + self._pad.barX) * i +
          (size.barWidth / 2);
      })
      .attr("y", function(d, i) {
        return self._pad.top + size.height + self._pad.bottom - 3;
      })
      .attr("width", size.barWidth)
      .attr("text-anchor", "middle")
      .text(function(d, i) {
        return _getValue(labelValue, d, i);
      });
    this._chart.append("line")
      .attr("class", "rule")
      .attr("y1", size.height + this._pad.top + 2)
      .attr("y2", size.height + this._pad.top + 2)
      .attr("x1", this._pad.left - 2)
      .attr("x2", this._pad.left + size.width + (size.bars * this._pad.barX));
  };

  BarChart.prototype._render_yLabel = function(entered, labelValue) {
    var size = this._getChartDimensions();
    var self = this;
    entered.append("text")
      .attr("class", "yLabel")
      .attr("text-anchor", "middle")
      .attr("width", size.barWidth)
      .attr("x", function(d, i) {
        return self._pad.left +
          (size.barWidth + self._pad.barX) * i +
          (size.barWidth / 2);
      })
      .attr("y", function(d, i) {
        console.log(size);
        return self._pad.top + (
          size.height - _getHeight(d, size.maxValue, size.height)) - 5;
      })
      .text(function(d, i) {
        return _getValue(labelValue, d, i);
      });
  };

  var _getHeight = function(value, maxValue, maxHeight) {
    var height = (value / maxValue) * maxHeight;
    return height;
  };

  var _getValue = function(getValue, point, index) {
    if (!$.isFunction(getValue)) {
      return getValue;
    }
    return getValue(point, index);
  };

  window.Charts = {
    BarChart: BarChart
  };

})();

var JULY = JULY || {};

JULY.Commit = Backbone.Model.extend({
  // Commit model
  url: '/api/v1/commit/'
});

JULY.CommitCollection = Backbone.Collection.extend({
  model: JULY.Commit,

  url: function() {return '/api/v1/commit/?' + this.params();},

  initialize: function(data, options) {
    this.projectId = options.projectId;
    this.userId = options.userId;
    this.limit = options.limit || 20;
    this.offset = options.offset || 0;
    this.total = 0;
    this.hasMore = false;
    this._pushStream = new PushStream({
      host: window.location.hostname,
      port: window.location.port,
      modes: "websocket|stream",
      urlPrefixStream: "/events/sub",
      urlPrefixPublisher: "/events/pub",
      urlPrefixWebsocket: "/events/ws"
    });
    JULY.collection = this;
    this._pushStream.onmessage = function(text) {
      console.log("-- New commit from PushStream:", text);
      JULY.collection.unshift(text);
    };
    this._pushStream.onstatuschange = function(state) {
      console.log("-- PushStream state changed: " + state);
      if (state === PushStream.CLOSED) {
        console.log("!! ERROR: PushStream was closed.");
      }
    };
    var channel = (this.projectId) ? "project-" + this.projectId : (this.userId) ? "user-" + this.userId: "global";
    console.log("-- Subscribing to PushStream channel: " + channel);
    this._pushStream.addChannel(channel);
    this._pushStream.connect();
  },

  params: function() {
    var p = {
      limit: this.limit,
      offset: this.offset
    };
    if (this.projectId) {
      p.project = this.projectId;
    }
    if (this.userId) {
      p.user = this.userId;
    }
    return jQuery.param(p);
  },

  parse: function(resp) {
    this.total = resp.meta.total_count;
    this.offset = resp.meta.offset + this.limit;
    this.hasMore = this.total > this.models.length;
    return resp.objects;
  }

});

JULY.CommitsView = JULY.ViewModel.extend({

  initialize: function(options) {
    this.c = new JULY.CommitCollection(null, options);
    this.c.fetch({add: true});
    this.commits = kb.collectionObservable(this.c);
  },

  hasMore: function() {
    return this.commits.collection().hasMore;
  },

  scrolled: function(data, event) {
        var elem = event.target;
        if (elem.scrollTop > (elem.scrollHeight - elem.offsetHeight - 200)) {
            this.fetch();
        }
    },

  fetch: function(){
    if (this.hasMore()) {
      this.commits.collection().fetch({add:true});
    }
  }
});


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
      p['name__icontains'] = this.query;
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
