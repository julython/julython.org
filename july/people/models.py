from google.appengine.ext import db
import logging

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
    timestamp = db.DateTimeProperty(auto_now_add=True)
    
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
