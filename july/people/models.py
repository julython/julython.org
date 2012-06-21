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
    def make_key(cls, commit_hash, user=None):
        """
        Return a ndb.Key for this commit, if the user is passed
        make the user the parent of the commit.
        """
        if user and isinstance(user, ndb.Model):
            return ndb.Key('User', user.key.id(), 'Commit', commit_hash)
        elif user and isinstance(user, ndb.Key):
            return ndb.Key('User', user.id(), 'Commit', commit_hash)
        else:
            return ndb.Key('Commit', commit_hash)
    
    @classmethod
    def create_by_email(cls, email, commits, project=None):
        """Create a commit by email address"""
        return cls.create_by_auth_id('email:%s' % email, commits, project)
    
    @classmethod
    def create_by_auth_id(cls, auth_id, commits, project=None):
        user = User.get_by_auth_id(auth_id)
        if user:
            return cls.create_by_user(user, commits, project)
        return cls.create_orphan(commits)
    
    @classmethod
    def create_by_user(cls, user, commits, project=None):
        """Create a commit with parent user, updating users points."""
        if not isinstance(commits, (list, tuple)):
            commits = [commits]
        user_key = user.key
        
        @ndb.transactional
        def txn():
            to_put = []
            for c in commits:
                commit_hash = c.get('hash')
                if commit_hash is None:
                    logging.info("Commit hash missing in create.")
                    continue
                commit_key = cls.make_key(commit_hash, user=user_key)
                commit = commit_key.get()
                if commit is None:
                    commit = Commit(key=commit_key, **c)
                    to_put.append(commit)
            
            # Check if there are no new commits and return
            if len(to_put) == 0:
                return
            
            user = user_key.get()
            
            if project is not None:
                # get the list of existing projects and make it a set 
                # to filter uniques
                projects = set(getattr(user, 'projects', []))
                projects.add(project)
                user.projects = list(projects)

            count = getattr(user, 'total', 0)
            user.total = count + len(to_put)
            to_put.append(user)
            keys = ndb.put_multi(to_put)
            return keys
        
        commits = txn()
        return commits
    
    @classmethod
    def create_orphan(cls, commits):
        """Create a commit with no parent."""
        if not isinstance(commits, (list, tuple)):
            commits = [commits]
        
        to_put = []
        for c in commits:
            commit_hash = c.get('hash')
            if commit_hash is None:
                logging.info("Commit hash missing in create.")
                continue
            commit_key = cls.make_key(commit_hash)
            commit = commit_key.get()
            if commit is None:
                commit = Commit(key=commit_key, **c)
                to_put.append(commit)
        
        commits = ndb.put_multi(to_put)
        return filter(lambda x: x.kind() != 'User', commits)
    
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
    name = ndb.StringProperty(required=False)
    forked = ndb.BooleanProperty(default=False)
    parent_url = ndb.StringProperty(required=False)
    created_on = ndb.DateTimeProperty(auto_now_add=True)
    # new projects start off with 10 points
    total = ndb.IntegerProperty(default=10)
    
    @property
    def project_name(self):
        return self.parse_project_name(self.url)
    
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
    
    @classmethod
    def get_or_create(cls, **kwargs):
        url = kwargs.pop('url', None)
        if url is None:
            raise AttributeError('Missing url in project create')
        
        created = False
        key = cls.make_key(url)
        project = key.get()
        if project is None:
            created = True
            project = Project(key=key, **kwargs)
            project.put()
        
        return created, project