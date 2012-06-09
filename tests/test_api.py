import sys
import unittest
import json
import webtest
import os
import logging

from google.appengine.api import memcache
from google.appengine.ext import db
from google.appengine.ext import testbed

os.environ['DJANGO_SETTINGS_MODULE'] = 'july.settings'

def setup_paths():
    """Setup sys.path with everything we need to run."""
    import google
    
    DIR_PATH = os.path.abspath(os.path.dirname(os.path.dirname(google.__file__)))
    
    EXTRA_PATHS = [
      os.getcwd(),
      DIR_PATH,
      os.path.join(DIR_PATH, 'lib', 'antlr3'),
      os.path.join(DIR_PATH, 'lib', 'django_1_3'),
      os.path.join(DIR_PATH, 'lib', 'fancy_urllib'),
      os.path.join(DIR_PATH, 'lib', 'ipaddr'),
      os.path.join(DIR_PATH, 'lib', 'jinja2'),
      os.path.join(DIR_PATH, 'lib', 'protorpc'),
      os.path.join(DIR_PATH, 'lib', 'markupsafe'),
      os.path.join(DIR_PATH, 'lib', 'webob_0_9'),
      os.path.join(DIR_PATH, 'lib', 'webapp2'),
      os.path.join(DIR_PATH, 'lib', 'yaml', 'lib'),
      os.path.join(DIR_PATH, 'lib', 'simplejson'),
      os.path.join(DIR_PATH, 'lib', 'google.appengine._internal.graphy'),
    ]
    
    sys.path = EXTRA_PATHS + sys.path

setup_paths()

from gae_django.auth.models import User
from api import app, make_digest
from july.people.models import Commit

class WebTestCase(unittest.TestCase):
    
    APPLICATION = None
    
    def setUp(self):
        # First, create an instance of the Testbed class.
        self.testbed = testbed.Testbed()
        # Then activate the testbed, which prepares the service stubs for use.
        self.testbed.activate()
        # Next, declare which service stubs you want to use.
        self.testbed.init_datastore_v3_stub()
        self.testbed.init_memcache_stub()
        if self.APPLICATION:
            self.app = webtest.TestApp(self.APPLICATION)

    def tearDown(self):
        self.testbed.deactivate()
    
    def make_user(self, username, save=True, **kwargs):
        user = User(username=username, **kwargs)
        if save:
            user.put()
        return user
    
    def make_commit(self, user, message, save=True, **kwargs):
        commit = Commit(parent=user, message=message, **kwargs)
        if save:
            commit.put()
        return commit

class ApiTests(WebTestCase):
    
    APPLICATION = app
    
    def test_commit_post(self):
        self.make_user('josh', email='josh@example.com')
        body = { "name": "Josh Marshall", 
            "email": "josh@example.com", 
            "message": "Working on Tornado stuff!", 
            "url": "https://github.com/project/commitID", 
            "timestamp": 5430604985.0,
            "hash": "6a87af2a7eb3de1e17ac1cce41e060516b38c0e9"}
        body_json = json.dumps(body)
        resp = self.app.post('/api/v1/commits', body_json, headers={"Authorization": make_digest('josh')})
        self.assertEqual(resp.status_code, 201)

    def test_get_commits(self):
        user = self.make_user(username="josh")
        self.make_commit(user, message='Working on the pythons')
        self.make_commit(user, message="Still working on the pythons")
        resp = self.app.get('/api/v1/commits')
        message = json.loads(resp.body)
        self.assertEqual(len(message['models']), 2)
    
    def test_missing_auth(self):
        resp = self.app.post('/api/v1/commits', 'body-text-here', status=401)
        self.assertEqual(resp.status_code, 401)
    
    def test_malformed_post(self):
        self.make_user('josh', email='josh@example.com')
        # post missing required field
        body = { "name": "Josh Marshall"}
        body_json = json.dumps(body)
        resp = self.app.post('/api/v1/commits', body_json, headers={"Authorization": make_digest('josh')}, status=400)
        self.assertEqual(resp.status_code, 400)
    
    def test_bad_user(self):
        body = { "message": "Josh Marshall"}
        body_json = json.dumps(body)
        resp = self.app.post('/api/v1/commits', body_json, headers={"Authorization": make_digest('josh')}, status=403)
        self.assertEqual(resp.status_code, 403)
    
    def test_data_saved_after_post(self):
        self.make_user('josh', email='josh@example.com')
        body = { "name": "Josh Marshall", 
            "email": "josh@example.com", 
            "message": "Working on Tornado stuff!", 
            "url": "https://github.com/project/commitID", 
            "timestamp": 5430604985.0,
            "hash": "6a87af2a7eb3de1e17ac1cce41e060516b38c0e9"}
        body_json = json.dumps(body)
        resp = self.app.post('/api/v1/commits', body_json, headers={"Authorization": make_digest('josh')})
        resp_body = json.loads(resp.body)
        
        # try to get the commit from the api
        resp = self.app.get('/api/v1/commits/%s' % resp_body['commit'])
        self.assertEqual(resp.status_code, 200)
    
    def test_commit_data_from_post(self):
        self.make_user('josh', email='josh@example.com')
        body = { "name": "Josh Marshall", 
            "email": "josh@example.com", 
            "message": "Working on Tornado stuff!", 
            "url": "https://github.com/project/commitID", 
            "timestamp": 5430604985.0,
            "hash": "6a87af2a7eb3de1e17ac1cce41e060516b38c0e9"}
        body_json = json.dumps(body)
        resp = self.app.post('/api/v1/commits', body_json, headers={"Authorization": make_digest('josh')})
        resp_body = json.loads(resp.body)
        
        # try to get the commit from the api
        resp = self.app.get('/api/v1/commits/%s' % resp_body['commit'])
        commit = json.loads(resp.body)
        self.assertEqual(commit['name'], "Josh Marshall")
        self.assertEqual(commit['email'], "josh@example.com")
        self.assertEqual(commit['message'], "Working on Tornado stuff!")
        self.assertEqual(commit['url'], "https://github.com/project/commitID")
        self.assertEqual(commit['timestamp'], '5430604985.0')
        self.assertEqual(commit['hash'], "6a87af2a7eb3de1e17ac1cce41e060516b38c0e9")
    
    def test_commit_not_found(self):
        resp = self.app.get('/api/v1/commits/blah', status=404)
        self.assertEqual(resp.status_code, 404)
    

if __name__ == '__main__':
    logging.getLogger().setLevel(logging.DEBUG)
    unittest.main()
      
    