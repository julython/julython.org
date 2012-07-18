
import datetime
import json

from google.appengine.ext import deferred
from google.appengine.ext.ndb import model
from google.appengine.api import channel

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
    
    def to_json(self):
        data = self.to_dict(exclude=['timestamp'])
        return json.dumps(data)
    
    @classmethod
    def create_message(cls, username, picture_url, message, **kwargs):
        message = cls(username=username, picture_url=picture_url, message=message)
        message.populate(**kwargs)
        message.put()
        
        deferred.defer(send_live_message, message.key.urlsafe(), _queue="live")
        return message
    
    @classmethod
    def add_commit(cls, key):
        commit_key = model.Key(urlsafe=key)
        commit = commit_key.get()
        parent_key = commit_key.parent()
        if parent_key is None:
            return
        
        parent = parent_key.get()
        
        picture_url = '/static/images/spread_the_word_button.png'
        if hasattr(parent.picture_url):
            picture_url = parent.picture_url
        
        message = cls(username=parent.username, 
            picture_url=picture_url, message=commit.message[:200], url=commit.url,
            project=commit.project, commit_hash=commit.hash)
        message.put()
        
        deferred.defer(send_live_message, message.key.urlsafe(), _queue="live")
        return message

class Connection(model.Model):
    """Store all the connected clients."""
    
    client_id = model.StringProperty()
    timestamp = model.DateTimeProperty(auto_now=True)
    

def send_live_message(key):
    """Deferred task for sending messages to the open channels."""
    
    message_key = model.Key(urlsafe=key)
    message = message_key.get()
    
    if message is None:
        return
    
    # Only notify rescent connections
    timestamp = datetime.datetime.now()
    oldest = timestamp - datetime.timedelta(hours=2)
    
    connections = Connection.query().filter(Connection.timestamp >= oldest).fetch(200, keys_only=True)
    
    for connection in connections:
        channel.send_message(connection.id(), message.to_json())