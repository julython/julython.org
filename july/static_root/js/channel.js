(function() {

  var Channel = function(el, token, options) {
    this._el = $(el);
    this._options = options || {};
    this._token = token;
    //console.log(this);
    //console.log(this._setupOptions);
    this._setupOptions();
    this._connect();
  };

  Channel.prototype._setupOptions = function() {
    this._options.maxMessages = this._options.maxMessages || 100;
    this._totalEl = this._options.totalEl;
    this._total = parseInt(this._options.total);
  };

  Channel.prototype._connect = function() {
    var self = this;
    var channel = new goog.appengine.Channel(this._token);
    var socket = channel.open();
    socket.onmessage = function(message) {
        self._newMessage(message);
    };
    socket.onerror = function(message){
        console.log(message);
    };
  };

  Channel.prototype.prepopulate = function(messages) {
    messages = JSON.parse(messages);
    for (var i=0; i<messages.length; ++i) {
      this._newMessage(messages[i]);
    }
  };

  Channel.prototype._newMessage = function(message) {
    var self = this;
    var messageView = this._buildMessageView(message, self);
    messageView.hide();
    this._el.prepend(messageView);
    messageView.slideDown();
    while (this._el.children().length > this._options.maxMessages) {
      this._el.children().last().remove();
    }
    if (this._total) {
        $(this._totalEl).html(this._total);
    }
  };

  Channel.prototype._buildMessageView = function(message, self) {
    // should move to a template eventually...
    if (message.data && typeof(message.data) === "string") {
      // it's a JSON message from Google.
      message = JSON.parse(message.data);
      if (message.commit_hash) {
        self._total += 1;
      }
    }
    var li = $('<li class="message"></li>');
    li.append('<img src="'+message.picture_url+'" class="profile-image"/>');
    li.append('<h4 class="username">'+message.username+'</h4>');
    if (message.project) {
        li.append('<p><a target="_blank" href="'+message.project+'">'+message.project+'</a></p>');
    }
    if (message.url) {
      var p = $('<p class="message-content"></p>')
      p.append('<a target="_blank" href="'+ message.url +'">'+message.message+'</a>')
    } else {
      var p = $('<p class="message-content"></p>');   
      p.text(message.message);
    }
    li.append(p);
    return li;
  };

  Channel.prototype._startDevMessages = function() {
    var self = this;
    this._interval = window.setInterval(function() {
      self._newMessage({
        user: "Joe User",
        image: "/static/images/participating_button.png",
        message: "This is a new message!"
      });
    }, 1200);
  };

  if (!window.julython) {
    window.julython = {};
  }

  window.julython.Channel = Channel;

})();
