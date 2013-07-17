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
    },
    update: function(element, valueAccessor, allBindingsAccessor) {
        var value = valueAccessor();
        allBindingsAccessor();
        var valueUnwrapped = ko.utils.unwrapObservable(value);
        $(element).timeago('update', valueUnwrapped);
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

JULY.csrfSafeMethod = function(method) {
    // these HTTP methods do not require CSRF protection
    return (/^(GET|HEAD|OPTIONS|TRACE)$/.test(method));
};

JULY.setCSRFToken = function(csrftoken) {
  jQuery.ajaxSetup({
    crossDomain: false, // obviates need for sameOrigin test
    beforeSend: function(xhr, settings) {
      if (!JULY.csrfSafeMethod(settings.type)) {
        xhr.setRequestHeader("X-CSRFToken", csrftoken);
      }
    }
  });
};

$(document).ready(function(){
    var $send_abuse = $('#send-abuse');
    var $form = $('#abuse-form');
    var $modal = $('#abuse-modal');
    var $abuseli = $('#abuseli');
    $send_abuse.click(function(){
        var desc = $form.find('textarea').val();
        if(!desc) return false;
        $.ajax({
            type: $form.attr('method'),
            url: $form.attr('action'),
            data: $form.serialize()
        });
        $modal.find('.modal-footer').html('');
        $modal.find('.modal-body').html('<h4>Thank you !</h4>');
        $send_abuse.remove();
        $abuseli.remove();
        return false;
    })
});
