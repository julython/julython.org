import logging
from urlparse import urlparse

from google.appengine.ext import ndb
from gae_django.auth.models import User

class Commit(ndb.Model):
    """
    Commit record for the profile, the parent is the profile
    that way we can update the commit count and last commit timestamp
    in the same transaction.
    """
    hash = ndb.StringProperty()
    author = ndb.StringProperty(indexed=False)
    name = ndb.StringProperty(indexed=False)
    email = ndb.StringProperty()
    message = ndb.StringProperty(indexed=False)
    url = ndb.StringProperty()
    timestamp = ndb.StringProperty()
    created_on = ndb.DateTimeProperty(auto_now_add=True)
    
    @classmethod
    def create_by_email(cls, email, data):
        """Create a commit by email address"""
        return cls.create_by_auth_id('email:%s' % email, data)
    
    @classmethod
    def create_by_auth_id(cls, auth_id, data):
        user = User.get_by_auth_id(auth_id)
        if user:
            return cls.create_by_user(user, data)
        return cls.create_orphan(data)
    
    @classmethod
    def create_by_user(cls, user, data):
        """Create a commit with parent user, updating users points."""
        user_key = user.key
        
        @ndb.transactional
        def txn():
            user = user_key.get()
            commit = Commit(parent=user_key, **data)
            count = getattr(user, 'total', 0)
            user.total = count + 1
            ndb.put_multi([commit, user])
            return commit
        
        commit = txn()
        return commit
    
    @classmethod
    def create_orphan(cls, data):
        """Create a commit with no parent."""
        commit = Commit(**data)
        commit.put()
        return commit
    
class Project(ndb.Model):
    """
    Project Model:
    
    This is either a brand new project or an already existing project
    such as #django, #fabric, #tornado, #pip, etc. 
    
    When a user Tweets a url we can automatically create anew project
    for any of the repo host we know already. (github, bitbucket)
    """
    
    url = ndb.StringProperty()
    description = ndb.TextProperty(required=False)
    forked = ndb.BooleanProperty(default=False)
    parent_url = ndb.StringProperty(required=False)
    created_on = ndb.DateTimeProperty(auto_now_add=True)
    # new projects start off with 10 points
    total_points = ndb.IntegerProperty(default=10)
    
    @classmethod
    def parse_project_name(cls, url):
        """
        Parse a project url and return a name for it.
        
        Example::
        
            Given:
              http://github.com/julython/julython.org
            Return:
              gh-julython-julython.org
        
        This is used as the Key name in order to speed lookups during
        api requests.
        """
        hosts_lookup = {
            'github.com': 'gh',
            'bitbucket.org': 'bb',
        }
        parsed = urlparse(url)
        path = parsed.path
        if path.startswith('/'):
            path = path[1:]
        tokens = path.split('/')
        host_abbr = hosts_lookup.get(parsed.netloc, 'o')
        name = '-'.join(tokens)
        return '%s-%s' % (host_abbr, name)
    
    @classmethod
    def make_key(cls, url):
        """Return a ndb.Key from a url."""
        name = cls.parse_project_name(url)
        return ndb.Key('Project', name)