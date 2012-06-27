import logging
from urlparse import urlparse

from google.appengine.ext import ndb
from google.appengine.ext import deferred
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
    project = ndb.StringProperty()
    project_slug = ndb.ComputedProperty(lambda self: Project.parse_project_name(self.project))
    timestamp = ndb.DateTimeProperty()
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
        return cls.create_by_auth_id('email:%s' % email, commits, project=project)
    
    @classmethod
    def create_by_auth_id(cls, auth_id, commits, project=None):
        user = User.get_by_auth_id(auth_id)
        if user:
            return cls.create_by_user(user, commits, project=project)
        return cls.create_orphan(commits, project=project)
    
    @classmethod
    def create_by_user(cls, user, commits, project=None):
        """Create a commit with parent user, updating users points."""
        if not isinstance(commits, (list, tuple)):
            commits = [commits]
        user_key = user.key
        project = project
        logging.info(project)
        
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
                    c['project'] = project
                    commit = Commit(key=commit_key, **c)
                    to_put.append(commit)
            
            # Check if there are no new commits and return
            if len(to_put) == 0:
                return
            
            user = user_key.get()
            count = getattr(user, 'total', 0)
            
            if project is not None:
                # get the list of existing projects and make it a set 
                # to filter uniques, if this project is new add it and 
                # update the users total points.
                projects = set(getattr(user, 'projects', []))
                if project not in projects:
                    logging.info('Adding project to user: %s', user)
                    projects.add(project)
                    count += 10
                    user.projects = list(projects)

            points = len(to_put)
            
            # defer updating the users location if they have one.
            if user.location:
                logging.info("deferring add points to location: %s", user.location_slug)
                deferred.defer(add_points_to_location, user.location_slug, points, project)
            
            user.total = count + points
            to_put.append(user)
            keys = ndb.put_multi(to_put)
            return keys
        
        commits = txn()
        return filter(lambda x: x.kind() == 'Commit', commits)
    
    @classmethod
    def create_orphan(cls, commits, project=None):
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
                c['project'] = project
                commit = Commit(key=commit_key, **c)
                to_put.append(commit)
        
        commits = ndb.put_multi(to_put)
        return commits
    
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
    forks = ndb.IntegerProperty(default=0)
    watchers = ndb.IntegerProperty(default=0)
    parent_url = ndb.StringProperty(required=False)
    created_on = ndb.DateTimeProperty(auto_now_add=True)
    # new projects start off with 10 points
    total = ndb.IntegerProperty(default=10)
    
    def __str__(self):
        if self.name:
            return self.name
        else:
            return self.url
    
    def __unicode__(self):
        return self.__str__()
    
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
        if not url:
            return
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
        url = kwargs.get('url', None)
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

class Location(ndb.Model):
    """Simple model for holding point totals and projects for a location"""
    total = ndb.IntegerProperty(default=0)
    projects = ndb.StringProperty(repeated=True)
    
    def __str__(self):
        return self.key.id().replace('-', ' ')
    
    def __unicode__(self):
        return self.__str__()
    
    @property
    def slug(self):
        return self.key.id()
    
def add_points_to_location(slug, points, project_url=None):
    """Add points and project_url to a location by slug.
    
    This is a simple method that runs in a transaction to 
    get or insert the location model, then update the points.
    It is meant to be run as a deferred task like so::
    
        from google.appengine.ext import deferred
        
        deferred.defer(add_points_to_location, 'some-slug-text', 10, 
            'http://github.com/my/project')
        
        # or without a project
        deferred.defer(add_points_to_location, 'some-slug-text', 3)
    """
    
    @ndb.transactional
    def txn(slug, points, project):
        location = Location.get_or_insert(slug)
        if project is not None and project not in location.projects:
            location.projects.append(project)
            points += 10
            
        location.total += points
        location.put()
        return location
    
    return txn(slug, points, project_url)
        
        