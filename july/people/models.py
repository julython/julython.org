import logging
from urlparse import urlparse

from google.appengine.ext import db

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
    
class Project(db.Model):
    """
    Project Model:
    
    This is either a brand new project or an already existing project
    such as #django, #fabric, #tornado, #pip, etc. 
    
    When a user Tweets a url we can automatically create anew project
    for any of the repo host we know already. (github, bitbucket)
    """
    
    url = db.URLProperty()
    description = db.TextProperty(required=False)
    forked = db.BooleanProperty(default=False)
    parent_url = db.URLProperty(required=False)
    created_on = db.DateTimeProperty(auto_now_add=True)
    # new projects start off with 10 points
    total_points = db.IntegerProperty(default=10)
    
    @classmethod
    def parse_project_name(cls, url):
        """
        Parse a project url and return a name for it.
        
        Example::
        
            Given:
              http://github.com/julython/julython.org
            Return:
              julython-julython-org
        
        This is used as the Key name in order to speed lookups during
        api requests.
        """
        logging.debug("Parsing url: %s", url)
        parsed = urlparse(url)
        path = parsed.path
        if path.startswith('/'):
            path = path[1:]
        tokens = path.split('/')
        return '-'.join(tokens)