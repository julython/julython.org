from google.appengine.ext import db
from gae_django.auth.models import User

class Record(db.Model):
    """
    A message from twitter or other api endpoint.
    
    Parent is the user who sent the message.
    """
    
    message = db.StringProperty()
    created_at = db.DateTimeProperty(auto_now_add=True)
    # denormalize the user data 
    username = db.StringProperty(indexed=False)
    display_name = db.StringProperty(indexed=False)
    
    def __str__(self):
        return self.message
    
    def __unicode__(self):
        return self.message
    
    @classmethod
    def create(cls, username, message):
        """
        Create a record for the User and safely update 
        their message count in a transaction. This will
        also fire off the task queue to parse the message
        for projects and other related data.
        """
        if not isinstance(username, basestring):
            return

        def txn(username, message):
            # Create the message with the user as the parent.
            # Update the message count as well.
            user = User.all().filter('username', username)
            if user is None:
                return
            
            # The user model is an Expando model, so it may not have 
            # a message_count property, look before you measure it.
            initial = 0
            if hasattr(user, 'message_count'):
                initial = user.message_count
            
            user.message_count = initial + 1
                
            record = cls(
                parent=user.key(),
                message=message,
                username=user.username,
                display_name=str(username))
            
            db.put([user, record])
            
            return record

        record = db.run_in_transaction(txn, username, message)
        

class Project(db.Model):
    """Simple project model which can be the target of a tweet. for example::
        
        john_doe: Just started #superproject http://bit.ly/lkjhasdf #julython
    
    The project name in this case will be 'superproject'
    The project repo will be parsed to find the long url for the shortened link.
    
    Bug Fixes::
        
        john_doe: Just fixed #145 #superproject @julython
    
    Tweets with the keywords 'fixed' or 'fixed bug' followed by #something
    'something' will be ingored and point awarded to any other hashtag.
    
    Forks::
    
        john_doe: Forked #django for fun #julython
        john_doe: Forked #django http://bit.ly/lkjhasdf @julython
    
    Points:
      * 10 points for new project
      * 10 points for bug fix:
    """
    name = db.StringProperty()
    total = db.IntegerProperty(default=0)
    repo = db.URLProperty()
    # Was this created during julython? if so we can track this @ github
    new_project = db.BooleanProperty(default=False)