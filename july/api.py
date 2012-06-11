import webapp2
import logging
import json
import datetime
import time
import hmac
import hashlib

from google.appengine.ext import db

from july.pages.models import Section
from gae_django.auth.models import User

from july import settings

from webapp2 import abort
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
        elif isinstance(value, db.GeoPt):
            output[key] = {'lat': value.lat, 'lon': value.lon}
        elif isinstance(value, db.Model):
            output[key] = to_dict(value)
        else:
            raise ValueError('cannot encode property: %s', key)
        return output
    
    for key, prop in model.properties().iteritems():
        output = encode(output, key, model)
    
    if isinstance(model, db.Expando):
        for key in model.dynamic_properties():
            output = encode(output, key, model)

    return output

class API(webapp2.RequestHandler):
    
    endpoint = None
    model = None
    
    @webapp2.cached_property
    def user(self):
        """Check the authorization header for a username to lookup"""
        headers = self.request.headers
        auth = headers.get('Authorization', None)
        if auth:
            return verify_digest(auth)
        return None
    
    def base_url(self):
        return self.request.host_url + '/api/v1'
    
    def uri(self):
        if self.endpoint:
            return self.base_url() + self.endpoint
        return self.base_url()
    
    def resource_uri(self, model):
        return '%s/%s' % (self.uri(), model.key().id_or_name())
    
    def serialize(self, model):
        # Allow models to override the default to_dict
        if hasattr(self.model, 'to_dict'):
            resp = self.model.to_dict(model)
        else:
            resp = to_dict(model)
        resp['uri'] = self.resource_uri(model)
        resp['key'] = str(model.key())
        resp['id'] = model.key().id_or_name()
        return resp
    
    def fetch_models(self):
        limit = int(self.request.get('limit', 100))
        cursor = self.request.get('cursor')
        order = self.request.get('order')
        filter_string = self.request.get('filter')
        
        query = self.model.all()
        
        if cursor:
            query.with_cursor(cursor)
        
        if order:
            query.order(order)
        
        # filter is ignored for apis that don't define 'handle_filter'
        if filter_string and hasattr(self, 'handle_filter'):
            query = self.handle_filter(query, filter_string)
        
        models = [self.serialize(m) for m in query.fetch(limit)]
        resp = {
            'limit': limit,
            'filter': filter_string,
            'cursor': query.cursor(),
            'uri': self.uri(),
            'models': models,
        }
        if len(models) == limit:
            resp['next'] = self.uri() + '?limit=%s&cursor=%s&filter=%s' % (limit, query.cursor(), filter_string)
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
                },
                {
                    'comment': 'Front Page Sections',
                    'url': self.uri() + '/sections'
                }
            ]
        }
        return self.respond_json(resp)

class CommitCollection(API):
    
    endpoint = '/commits'
    model = Commit
    
    def resource_uri(self, model):
        return '%s/%s' % (self.uri(), model.key())
    
    def handle_filter(self, query, filter_string):
        """Allow for filtering by user or project"""
        if filter_string.startswith('#'):
            # we're looking for a project strip the hash tag first
            project_name = filter_string[1:]
            logging.error('Handle Projects!!! %s', project_name)
            raise
        elif filter_string.startswith('@'):
            username = filter_string[1:]
        else:
            username = filter_string
            
        logging.info('looking up commits for user: %s', username)
        user = User.all().filter('username', username).get()
        if user is None:
            abort(404)
        
        return query.ancestor(user)
    
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
        if self.user is None:
            abort(401)
        
        # check to see if the user is registered.
        user = User.all().filter('username', self.user).get()
        if not user:
            abort(403)
        
        # see how we were posted
        try:
            data = json.loads(self.request.body)
        except:
            abort(400)
            
        # create the new commit object
        form = CommitForm(data)
        if not form.is_valid():
            abort(400)
        
        def txn(user_key, data):
            user = db.get(user_key)
            commit = Commit(parent=user_key, **data)
            count = getattr(user, 'total', 0)
            user.total = count + 1
            db.put([commit, user])
            return commit
        
        commit = db.run_in_transaction(txn, user.key(), form.cleaned_data)
        
        self.respond_json({'commit': str(commit.key())}, status_code=201)
    
    def get(self):
        return self.fetch_models()

class CommitResource(API):
    
    endpoint = '/commits'
    model = Commit
    
    def get(self, commit_key):
        # Test if the string is an actual datastore key first
        try:
            key = db.Key(commit_key)
        except:
            abort(404)
            
        commit = db.get(key)
        if commit is None:
            abort(404)
        return self.respond_json(self.serialize(commit))

class SectionCollection(API):
    
    endpoint = '/sections'
    model = Section
    
    def get(self):
        return self.fetch_models()
        
class SectionResource(API):
    
    endpoint = '/sections'
    model = Section
    
    def get(self, section_id):
        instance = self.model.get_by_id(int(section_id))
        if instance is None:
            abort(404)
        return self.respond_json(self.serialize(instance))
            
class ProjectCollection(API):
    
    endpoint = '/projects'
    model = Project
    
    def get(self):
        return self.fetch_models()
    
    def post(self):
        """Create a project from the api.
        
        Example::
        
            {"url": "http://github.com/user/project",
             "forked": true,
             "parent_url": "http://github.com/other/project"}
        """
        if self.user is None:
            abort(401)
        
        # check to see if the user is registered.
        user = User.all().filter('username', self.user).get()
        if not user:
            abort(403)
        
        # see how we were posted
        try:
            data = json.loads(self.request.body)
        except:
            abort(400)
            
        # create the new commit object
        form = ProjectForm(data)
        if not form.is_valid():
            abort(400)
        
        post_data = {}
        for k, v in data.iteritems():
            if not v:
                continue
            post_data[k] = v
             
        created = False
        key = db.Key.from_path('Project', Project.parse_project_name(data['url']))
        project = db.get(key)
        
        if project is None:
            created = True
            project = Project(key=key, **post_data)
            db.put(project)
            
        def txn(user_key):
            user = db.get(user_key)
            count = getattr(user, 'total', 0)
            user.total = count + 10
            db.put([user])
            return user
        
        if created:
            user = db.run_in_transaction(txn, user.key())
        
        self.respond_json({'project': self.serialize(project)}, status_code=201 if created else 200)

class ProjectResource(API):
    
    endpoint = '/project'
    model = Project
    
    def get(self, project_name):
        instance = self.model.get_by_key_name(project_name)
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
        instance = self.model.all().filter('username', username).get()
        if instance is None:
            abort(404)
        return self.respond_json(self.serialize(instance))

###
### Setup the routes for the API
###
routes = [
    webapp2.Route('/api/v1', RootHandler),
    webapp2.Route('/api/v1/commits', CommitCollection),
    webapp2.Route('/api/v1/commits/<commit_key:\w+>', CommitResource),
    webapp2.Route('/api/v1/sections', SectionCollection),
    webapp2.Route('/api/v1/sections/<section_id:\d+>', SectionResource),
    webapp2.Route('/api/v1/projects', ProjectCollection),
    webapp2.Route('/api/v1/projects/<project_name:[\w-]+>', ProjectResource),
    webapp2.Route('/api/v1/people', PeopleCollection),
    webapp2.Route('/api/v1/people/<username:[\w_-]+>', PeopleResource),
] 

# The Main Application
app = webapp2.WSGIApplication(routes)