(function() {

  var Channel = function(el, token, options) {
    this._el = $(el);
    this._options = options || {};
    this._token = token;
    console.log(this);
    console.log(this._setupOptions);
    this._setupOptions();
    this._connect();
  };

  Channel.prototype._setupOptions = function() {
    this._options.maxMessages = this._options.maxMessages || 100;
  };

  Channel.prototype._connect = function() {
    // pretending to connect
    console.log("We're connecting!");
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
    var messageView = this._buildMessageView(message);
    messageView.hide();
    this._el.prepend(messageView);
    messageView.slideDown();
    while (this._el.children().length > this._options.maxMessages) {
      this._el.children().last().remove();
    }
  };

  Channel.prototype._buildMessageView = function(message) {
    // should move to a template eventually...
    if (message.data && typeof(message.data) === "string") {
      // it's a JSON message from Google.
      message = JSON.parse(message.data);
    }
    var li = $('<li class="message"></li>');
    li.append('<img src="'+message.picture_url+'" class="profile-image"/>');
    li.append('<h4 class="username">'+message.username+'</h4>');
    var p = $('<p class="message-content"></p>');
    p.text(message.message);
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
