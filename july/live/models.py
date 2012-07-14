
from google.appengine.ext import deferred
from google.appengine.ext.ndb import model

from tasks import create_message

class Message(model.Model):
    
    username = model.StringProperty()
    picture_url = model.StringProperty()
    message = model.StringProperty()
    commit_hash = model.StringProperty(required=False)
    url = model.StringProperty(required=False)
    project = model.StringProperty(required=False)
    timestamp = model.DateTimeProperty(auto_now_add=True)
    
    def __str__(self):
        return '%s - %s' % (self.username, self.message)
    
    def __unicode__(self):
        return self.__str__()
    
    @classmethod
    def create_message(cls, username, picture_url, message, **kwargs):
        message = cls(username=username, picture_url=picture_url, message=message)
        message.populate(**kwargs)
        message.put()
        
        deferred.defer(create_message, message.key.urlsafe())
        return message