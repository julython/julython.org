from google.appengine.ext import db
import logging

class Commit(db.Model):
    """
    Commit record for the profile, the parent is the profile
    that way we can update the commit count and last commit timestamp
    in the same transaction.
    """
    hash = db.StringProperty()
    author = db.StringProperty(indexed=False)
    name = db.StringProperty(indexed=False)
    email = db.EmailProperty()
    message = db.StringProperty(indexed=False)
    url = db.StringProperty()
    timestamp = db.StringProperty()
    created_on = db.DateTimeProperty(auto_now_add=True)
    
