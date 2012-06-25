import webapp2
import logging
import json
import datetime
import time
import hmac
import hashlib
from functools import wraps
from collections import defaultdict

from google.appengine.ext import ndb
from google.appengine.datastore.datastore_query import Cursor
from webapp2 import abort

from gae_django.auth.models import User

from july import settings
from july.pages.models import Section
from july.people.forms import CommitForm, ProjectForm
from july.people.models import Commit, Project

SIMPLE_TYPES = (int, long, float, bool, dict, basestring, list)

def make_digest(message):
    """Somewhat secure way to encode the username for tweets by the client."""
    salt = str(int(time.time()))
    key = ':'.join([salt, settings.SECRET_KEY])
    m = hmac.new(key, message, hashlib.sha256).hexdigest()
    return ':'.join([salt, message, m])

def verify_digest(message):
    """Decode the digest from the request Auth Headers"""
    salt, user_name, digest = message.split(':')
    key = ':'.join([salt, settings.SECRET_KEY])
    m = hmac.new(key, user_name, hashlib.sha256).hexdigest()
    if m == digest:
        return user_name
    return None

def to_dict(model):
    """
    Stolen from stackoverflow: 
    http://stackoverflow.com/questions/1531501/json-serialization-of-google-app-engine-models
    """
    output = {}
    
    def encode(output, key, model):
        value = getattr(model, key)

        if value is None or isinstance(value, SIMPLE_TYPES):
            output[key] = value
        elif isinstance(value, datetime.date):
            # Convert date/datetime to ms-since-epoch ("new Date()").
            ms = time.mktime(value.utctimetuple())
            ms += getattr(value, 'microseconds', 0) / 1000
            output[key] = int(ms)
        elif isinstance(value, ndb.GeoPt):
            output[key] = {'lat': value.lat, 'lon': value.lon}
        elif isinstance(value, ndb.Model):
            output[key] = to_dict(value)
        else:
            raise ValueError('cannot encode property: %s', key)
        return output
    
    for key in model.to_dict().iterkeys():
        output = encode(output, key, model)
    
    if isinstance(model, ndb.Expando):
        for key in model._properties.iterkeys():
            output = encode(output, key, model)

    return output

def utcdatetime(timestamp):
    """
    Take a timestamp in the formats::
        
        2012-05-30 04:07:03+00:00
        2012-05-30T04:07:03-08:00
    
    And return a utc normalized timestamp to insert into the db.
    """
    from iso8601 import parse_date
    
    d = parse_date(timestamp)
    utc = d - d.utcoffset()
    utc = utc.replace(tzinfo=None)
    return utc

def decorated_func(login_required=True, registration_required=True):
    """Simple decorator to require login and or registration."""
    
    def decorator(func):
        func._login_require = login_required
        func._registration_required = registration_required
        
        @wraps(func)
        def wrapper(self, *args, **kwargs):
            if login_required:
                if self.auth is None:
                    abort(401)
                
            if registration_required:
                # check to see if the user is registered.
                user = User.get_by_auth_id(self.auth)
                if not user:
                    abort(403)

            return func(self, *args, **kwargs)
        return wrapper
    return decorator

# default wrapper    
auth_required = decorated_func()

class API(webapp2.RequestHandler):
    
    endpoint = None
    model = None
    form = None
    
    def options(self):
        """Be a good netizen citizen and return HTTP verbs allowed."""
        valid = ', '.join(webapp2._get_handler_methods(self))
        self.response.set_status(200)
        self.response.headers['Allow'] = valid
        return self.response.out
    
    @webapp2.cached_property
    def auth(self):
        """Check the authorization header for a username to lookup"""
        headers = self.request.headers
        auth = headers.get('Authorization', None)
        if auth:
            return verify_digest(auth)
        return None
    
    def parse_form(self, form=None):
        """Hook to run the validation on a form"""
        form = form or self.form
        if not form:
            raise AttributeError("Form object missing")
        
        # see how we were posted
        try:
            data = json.loads(self.request.body)
        except:
            # fall back to POST and GET args
            data = self.request.params

        return form(data)
        
    @webapp2.cached_property
    def user(self):
        """Check the authorization header for a username to lookup"""
        if self.auth:
            return User.get_by_auth_id(self.auth)
        return None
    
    def base_url(self):
        return self.request.host_url + '/api/v1'
    
    def uri(self):
        if self.endpoint:
            return self.base_url() + self.endpoint
        return self.base_url()
    
    def resource_uri(self, model):
        return '%s/%s' % (self.uri(), model.key.id())
    
    def serialize(self, model):
        # Allow models to override the default to_dict
        if hasattr(self.model, 'serialize'):
            resp = self.model.serialize(model)
        else:
            resp = to_dict(model)
        resp['uri'] = self.resource_uri(model)
        resp['key'] = str(model.key)
        resp['id'] = model.key.id()
        return resp
    
    def fetch_models(self):
        limit = int(self.request.get('limit', 100))
        cursor = self.request.get('cursor', None)
        order = self.request.get('order')
        filter_string = self.request.get('filter')
        
        query = self.model.query()
        
        if cursor:
            cursor = Cursor(urlsafe=cursor)
        
        # filter is ignored for apis that don't define 'handle_filter'
        if filter_string and hasattr(self, 'handle_filter'):
            query = self.handle_filter(query, filter_string)

        if order and hasattr(self, 'handle_order'):
            query = self.handle_order(query, order)
        
        models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
        
        resp = {
            'limit': limit,
            'filter': filter_string,
            'cursor': next_cursor.urlsafe(),
            'uri': self.uri(),
            'models': [self.serialize(m) for m in models],
        }
        if more:
            resp['next'] = self.uri() + '?limit=%s&cursor=%s&filter=%s' % (limit, next_cursor.urlsafe(), filter_string)
        return self.respond_json(resp)
    
    def respond_json(self, message, status_code=200):
        self.response.set_status(status_code)
        self.response.headers['Content-type'] = 'application/json'
        self.response.headers['Access-Control-Allow-Origin'] = '*'
        
        try:
            resp = json.dumps(message)
        except:
            self.response.set_status(500)
            resp = json.dumps({u'message': u'message not serializable'})
        
        return self.response.out.write(resp)
        

class RootHandler(API):
    
    def get(self):
        """Just dish out some helpful uri info."""
        # TODO: autogenerate this!
        logging.error(self.request.route)
        resp = {
            'comment': "Welcome to the www.julython.org API",
            'version': '1',
            'uri': self.uri(),
            'endpoints': [
                {
                    'comment': 'Commit data',
                    'uri': self.uri() + '/commits'
                },
                {
                    'comment': 'People of julython',
                    'uri': self.uri() + '/people'
                },
                {
                    'comment': 'Project in julython',
                    'uri': self.uri() + '/projects'
                }
            ]
        }
        return self.respond_json(resp)

class CommitCollection(API):
    
    endpoint = '/commits'
    model = Commit
    form = CommitForm
    
    def resource_uri(self, model):
        return '%s/%s' % (self.uri(), model.key.urlsafe())
    
    def handle_filter(self, query, filter_string):
        """Allow for filtering by user or project"""
        if filter_string.startswith('#'):
            # we're looking for a project strip the hash tag first
            project_name = filter_string[1:]
            logging.error('Handle Projects!!! %s', project_name)
            raise
        elif filter_string.startswith('@'):
            username = 'twitter:%s' % filter_string[1:]
        else:
            username = filter_string
            
        logging.info('looking up commits for user: %s', username)
        user = User.get_by_auth_id(username)
        if user is None:
            abort(404)
            
        return self.model.query(ancestor=user.key)
    
    @auth_required
    def post(self):
        """Create a tweet from the api.
        
        Example::
        
            { "name": "Josh Marshall", 
            "email": "catchjosh@gmail.com", 
            "message": "Working on Tornado stuff!", 
            "url": "https://github.com/project/commitID", 
            "timestamp": 5430604985.0,
            "hash": "6a87af2a7eb3de1e17ac1cce41e060516b38c0e9"}
        """
        form = self.parse_form()
        if not form.is_valid():
            return self.respond_json(form.errors, status_code=400)
        
        commits = Commit.create_by_user(self.user, form.cleaned_data)
        
        self.respond_json({'commits': [k.urlsafe() for k in commits]}, status_code=201)
    
    def get(self):
        return self.fetch_models()

class CommitResource(API):
    
    endpoint = '/commits'
    model = Commit
    
    def get(self, commit_key):
        # Test if the string is an actual datastore key first
        try:
            key = ndb.Key(urlsafe=commit_key)
        except:
            abort(404)
            
        commit = key.get()
        if commit is None:
            abort(404)
        return self.respond_json(self.serialize(commit))
            
class ProjectCollection(API):
    
    endpoint = '/projects'
    model = Project
    form = ProjectForm
    
    def options(self):
        return self.response.out()
    
    def get(self):
        return self.fetch_models()
    
    @auth_required
    def post(self):
        """Create a project from the api.
        
        Example::
        
            {
                "url": "http://github.com/defunkt/github",
                "name": "github",
                "description": "You're lookin' at it.",
                "watchers": 5,
                "forks": 2,
                "private": 1,
                "email": "chris@ozmm.org",
                "account": "twitter_name",
            },
        """
        form = self.parse_form()
        if not form.is_valid():
            return self.respond_json(form.errors, status_code=400)
        
        # Lookup the user by email or account
        email = form.cleaned_data.pop('email', None)
        account = form.cleaned_data.pop('account', None)
        user = None
        if email:
            user = User.get_by_auth_id('email:%s' % email)
        elif account:
            user = User.get_by_auth_id('twitter:%s' % account)
        
        created = False
        project_url = form.cleaned_data['url']
        project_key = Project.make_key(project_url)
        project = project_key.get()
        
        if project is None:
            created = True
            project = Project(key=project_key, **form.cleaned_data)
            project.put()
        
        @ndb.transactional    
        def txn():
            count = getattr(user, 'total', 0)
            projects = set(getattr(user, 'projects', []))
            user.total = count + 10
            user.projects = list(projects.add(project_url))
            user.put()
            return user
        
        if created and user:
            txn()
        
        self.respond_json({'project': self.serialize(project)}, status_code=201 if created else 200)

class ProjectResource(API):
    
    endpoint = '/project'
    model = Project
    
    def get(self, project_name):
        project_key = ndb.Key(self.model._get_kind(), project_name)
        instance = project_key.get()
        if instance is None:
            abort(404)
        return self.respond_json(self.serialize(instance))

class PeopleCollection(API):
    
    endpoint = '/people'
    model = User
    
    def resource_uri(self, model):
        return '%s/%s' % (self.uri(), model.username)
    
    def get(self):
        return self.fetch_models()
    
class PeopleResource(API):
    
    endpoint = '/people'
    model = User
    
    def get(self, username):
        auth_id = username
        
        if '@' in username:
            auth_id = 'email:%s' % username
        elif ':' not in username:
            # default to twitter lookup
            auth_id = 'twitter:%s' % username
            
        instance = self.model.get_by_auth_id(auth_id)
        if instance is None:
            abort(404)
        return self.respond_json(self.serialize(instance))

class PostCallbackHandler(API):

    def parse_commits(self, commits):
        """
        Takes a list of raw commit data and returns a dict of::
            
            {'email': [list of parsed commits]}
        
        """
        commit_dict = defaultdict(list)
        for k, v in [self._parse_commit(data) for data in commits]:
            commit_dict[k].append(v)
        
        return commit_dict
    
    def _parse_repo(self, repository):
        """Parse a repository."""
        raise NotImplementedError("Subclasses must define this")
    
    def _parse_commit(self, commit):
        """Parse a single commit."""
        raise NotImplementedError("Subclasses must define this")
    
    def parse_payload(self):
        """
        Hook for turning post data into payload.
        """
        payload = self.request.params.get('payload')
        return payload
        
    def get(self):
        """Display some useful info about POSTing to this resource."""
        return self.respond_json({'description': self.__doc__})
    
    def post(self):
        payload = self.parse_payload()
        logging.info(payload)
        if not payload:
            abort(400)
        
        try:
            data = json.loads(payload)
        except:
            logging.exception("Unable to serialize POST")
            abort(400)
        
        repository = data.get('repository', {})
        commit_data = data.get('commits', [])
        
        repo = self._parse_repo(repository)
        
        _, project = Project.get_or_create(**repo)
        project_key = project.key
        
        commit_dict = self.parse_commits(commit_data)
        total_commits = []
        for email, commits in commit_dict.iteritems():
            # TODO: run this in a task queue?
            cmts = Commit.create_by_email(email, commits, project=project.url)
            total_commits += cmts
        
        @ndb.transactional
        def txn():
            # TODO: run this in a task queue?
            total = len(total_commits)
            
            project = project_key.get()
            logging.info("adding %s points to %s", total, project)
            project.total += total
            project.put()
        
        status = 200
        if len(total_commits):
            txn()
            status = 201
        
        return self.respond_json({'commits': [c.urlsafe() for c in total_commits]}, status_code=status)
        
    
class BitbucketHandler(PostCallbackHandler):
    """
    Take a POST from bitbucket in the format::
    
        payload=>"{
            "canon_url": "https://bitbucket.org", 
            "commits": [
                {
                    "author": "marcus", 
                    "branch": "featureA", 
                    "files": [
                        {
                            "file": "somefile.py", 
                            "type": "modified"
                        }
                    ], 
                    "message": "Added some featureA things", 
                    "node": "d14d26a93fd2", 
                    "parents": [
                        "1b458191f31a"
                    ], 
                    "raw_author": "Marcus Bertrand <marcus@somedomain.com>", 
                    "raw_node": "d14d26a93fd28d3166fa81c0cd3b6f339bb95bfe", 
                    "revision": 3, 
                    "size": -1, 
                    "timestamp": "2012-05-30 06:07:03", 
                    "utctimestamp": "2012-05-30 04:07:03+00:00"
                }
            ], 
            "repository": {
                "absolute_url": "/marcus/project-x/", 
                "fork": false, 
                "is_private": true, 
                "name": "Project X", 
                "owner": "marcus", 
                "scm": "hg", 
                "slug": "project-x", 
                "website": ""
            }, 
            "user": "marcus"
        }"
    """
    #TODO: make this work

class GithubHandler(PostCallbackHandler):
    """
    Takes a POST response from github in the following format::
    
        payload=>"{
              "before": "5aef35982fb2d34e9d9d4502f6ede1072793222d",
              "repository": {
                "url": "http://github.com/defunkt/github",
                "name": "github",
                "description": "You're lookin' at it.",
                "watchers": 5,
                "forks": 2,
                "private": 1,
                "owner": {
                  "email": "chris@ozmm.org",
                  "name": "defunkt"
                }
              },
              "commits": [
                {
                  "id": "41a212ee83ca127e3c8cf465891ab7216a705f59",
                  "url": "http://github.com/defunkt/github/commit/41a212ee83ca127e3c8cf465891ab7216a705f59",
                  "author": {
                    "email": "chris@ozmm.org",
                    "name": "Chris Wanstrath"
                  },
                  "message": "okay i give in",
                  "timestamp": "2008-02-15T14:57:17-08:00",
                  "added": ["filepath.rb"]
                },
                {
                  "id": "de8251ff97ee194a289832576287d6f8ad74e3d0",
                  "url": "http://github.com/defunkt/github/commit/de8251ff97ee194a289832576287d6f8ad74e3d0",
                  "author": {
                    "email": "chris@ozmm.org",
                    "name": "Chris Wanstrath"
                  },
                  "message": "update pricing a tad",
                  "timestamp": "2008-02-15T14:36:34-08:00"
                }
              ],
              "after": "de8251ff97ee194a289832576287d6f8ad74e3d0",
              "ref": "refs/heads/master"
            }"
    """
    
    def _parse_repo(self, data):
        """Returns a dict suitable for creating a project."""
        if not isinstance(data, dict):
            raise AttributeError("Expected a dict object")
        
        return {
            'url': data['url'],
            'description': data.get('description', ''),
            'name': data.get('name'),
            'forks': data.get('forks'),
            'watchers': data.get('watchers')
        }
    
    
    def _parse_commit(self, data):
        """Return a tuple of (email, dict) to simplify commit creation.
        
        Raw commit data::
        
            {
              "id": "41a212ee83ca127e3c8cf465891ab7216a705f59",
              "url": "http://github.com/defunkt/github/commit/41a212ee83ca127e3c8cf465891ab7216a705f59",
              "author": {
                "email": "chris@ozmm.org",
                "name": "Chris Wanstrath"
              },
              "message": "okay i give in",
              "timestamp": "2008-02-15T14:57:17-08:00",
              "added": ["filepath.rb"]
            },
        """
        if not isinstance(data, dict):
            raise AttributeError("Expected a dict object")
        
        author = data.get('author', {})
        email = author.get('email', '')
        name = author.get('name', '')
        
        commit_data = {
            'hash': data['id'],
            'url': data['url'],
            'email': email,
            'name': name,
            'message': data['message'],
            'timestamp': utcdatetime(data['timestamp']),
        }
        return email, commit_data
    
    
        
        
        
###
### Setup the routes for the API
###
routes = [
    webapp2.Route('/api/v1', RootHandler),
    webapp2.Route('/api/v1/commits', CommitCollection),
    webapp2.Route('/api/v1/commits/<commit_key:\w+>', CommitResource),
    webapp2.Route('/api/v1/projects', ProjectCollection),
    webapp2.Route('/api/v1/projects/<project_name:[\w-]+>', ProjectResource),
    webapp2.Route('/api/v1/people', PeopleCollection),
    webapp2.Route('/api/v1/people/<username:[\w_-]+>', PeopleResource),
    webapp2.Route('/api/v1/github', GithubHandler),
    webapp2.Route('/api/v1/bitbucket', BitbucketHandler),
] 

# The Main Application
app = webapp2.WSGIApplication(routes)