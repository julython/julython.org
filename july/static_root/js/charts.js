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
