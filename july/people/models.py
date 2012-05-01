from google.appengine.ext import db
import logging

class Profile(db.Model):
    """
    Basic profile to optionally describe the stuff the 
    developer is hacking on.
    """
    user_id = db.IntegerProperty(required=True)
    name = db.StringProperty(required=True)
    my_url = db.URLProperty()
    about_me = db.TextProperty()
    where = db.StringProperty(indexed=False)

    twitter = db.StringProperty(indexed=False)
    show_twitter = db.BooleanProperty(default=False, indexed=False)

    facebook = db.StringProperty(indexed=False)
    show_facebook = db.BooleanProperty(default=False, indexed=False)
    
    last_commit = db.DateTimeProperty()
    
    commits = db.IntegerProperty()
    
    def __unicode__(self):
        return unicode(self.parent())
    
    @property
    def secret(self):
        """
        This is the secret key the user will add to identify 
        themselves with the api. We just use the key as
        this is unique and allows us to add the commits to the
        proper entity group. 
        
        This is not secure so don't use this for anything too
        important.
        """
        return str(self.key())
    
    @classmethod
    def get_or_create(cls, user):
        profile = cls.all().filter('user_id', user.key().id()).get()
        if profile is None:
            profile = cls(user_id=user.key().id(), name=unicode(user))
            if user.service == 'twitter':
                profile.twitter = user.username
            elif user.service == 'facebook':
                profile.facebook = user.username
            profile.put()
        return profile

class Commit(db.Model):
    """
    Commit record for the profile, the parent is the profile
    that way we can update the commit count and last commit timestamp
    in the same transaction.
    """
    commit_hash = db.StringProperty(indexed=False)
    author = db.StringProperty()
    message = db.StringProperty()
    remote = db.StringProperty()
    timestamp = db.DateTimeProperty()
    
    @classmethod
    def create_from_json(cls, json_msg):
        """Helper method to create the commit from the api."""
        import json
        try:
            msg = json.loads(json_msg)
        except:
            logging.exception("Unable to parse json")
        
        # Check that minimun fields are available
        required = ['secret', 'author', 'timestamp', 'commit_hash']
        if not all([msg.get(k) for k in required]):
            raise Exception
        
        # TODO: Create in transaction and update profile.