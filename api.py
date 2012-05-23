import webapp2
import logging
import json
import datetime
import time

from google.appengine.ext import db

from july.pages.models import Section

SIMPLE_TYPES = (int, long, float, bool, dict, basestring, list)

def to_dict(model):
    """
    Stolen from stackoverflow: 
    http://stackoverflow.com/questions/1531501/json-serialization-of-google-app-engine-models
    """
    output = {}

    for key, prop in model.properties().iteritems():
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
            raise ValueError('cannot encode ' + repr(prop))

    return output

class API(webapp2.RequestHandler):
    
    endpoint = None
    model = None
    
    def base_url(self):
        return self.request.host_url + '/api/v1'
    
    def uri(self):
        if self.endpoint:
            return self.base_url() + self.endpoint
        return self.base_url()
    
    def serialize(self, model):
        # Allow models to override the default to_dict
        if hasattr(self.model, 'to_dict'):
            resp = self.model.to_dict(model)
        else:
            resp = to_dict(model)
        resp['uri'] = '%s/%s' % (self.uri(), model.key().id_or_name())
        resp['key'] = str(model.key())
        resp['id'] = model.key().id_or_name()
        return resp
    
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
                    'comment': 'commit data',
                    'uri': self.uri() + '/commits'
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
    
    def get(self):
        return self.respond_json({'commits': []})

class CommitResource(API):
    
    endpoint = '/commits'

class SectionCollection(API):
    
    endpoint = '/sections'
    
    def get(self):
        limit = int(self.request.get('limit', 100))
        cursor = self.request.get('cursor')
        
        query = Section.all()
        
        if cursor:
            query.with_cursor(cursor)
        
        sections = [self.serialize(section) for section in query.fetch(limit)]
        resp = {
            'limit': limit,
            'cursor': query.cursor(),
            'uri': self.uri(),
            'sections': sections
        }
        if len(sections) == limit:
            resp['next'] = self.uri() + '?limit=%s&cursor=%s' % (limit, query.cursor())
        return self.respond_json(resp)
        
class SectionResource(API):
    
    endpoint = '/sections'
    model = Section
    
    def get(self, section_id):
        instance = self.model.get_by_id(int(section_id))
        return self.respond_json(self.serialize(instance))

routes = [
    ('/api/v1', RootHandler),
    ('/api/v1/commits', CommitCollection),
    ('/api/v1/commits/(\d+)', CommitResource),
    ('/api/v1/sections', SectionCollection),
    ('/api/v1/sections/(\d+)', SectionResource),
] 

app = webapp2.WSGIApplication(routes)