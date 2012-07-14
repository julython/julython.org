
import json

from google.appengine.ext import ndb

def create_message(key):
    """Deferred task for sending messages to the open channels."""
    
    message_key = ndb.Key(urlsafe=key)
    message = message_key.get()
    
    if message is None:
        return
    
    # TODO: for c in channels: spam channel...