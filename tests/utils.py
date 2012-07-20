import sys
import unittest
import json
import webtest
import os
import logging
import datetime
import base64

from google.appengine.ext import ndb, deferred
from google.appengine.ext import testbed
from google.appengine.datastore import datastore_stub_util

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
from july.api import app, make_digest, utcdatetime
from july.people.models import Commit, Project, Location
from july import settings

class WebTestCase(unittest.TestCase):
    
    APPLICATION = None
    
    def setUp(self):
        # First, create an instance of the Testbed class.
        self.testbed = testbed.Testbed()
        # Then activate the testbed, which prepares the service stubs for use.
        self.testbed.activate()
        # create a policy object
        self.policy = datastore_stub_util.PseudoRandomHRConsistencyPolicy(probability=1)
        # Next, declare which service stubs you want to use.
        self.testbed.init_datastore_v3_stub(consistency_policy=self.policy)
        self.testbed.init_memcache_stub()
        os.environ['HTTP_HOST'] = 'localhost'
        self.testbed.init_taskqueue_stub(_all_queues_valid=True)
        self.taskqueue_stub = self.testbed.get_stub(testbed.TASKQUEUE_SERVICE_NAME)
        
        # reset testing settings
        settings.TESTING = True
        
        # The test application
        if self.APPLICATION:
            self.app = webtest.TestApp(self.APPLICATION)

    def tearDown(self):
        self.testbed.deactivate()
    
    def make_user(self, username, save=True, **kwargs):
        _, user = User.create_user(auth_id='%s' % username, **kwargs)
        return user
    
    def make_commit(self, user, message, save=True, **kwargs):
        commit = Commit(parent=user.key, message=message, **kwargs)
        if save:
            commit.put()
        return commit
    
    def make_orphan(self, email, message, save=True, **kwargs):
        commit = Commit(email=email, message=message, **kwargs)
        if save:
            commit.put()
        return commit
    
    def make_project(self, name, save=True, **kwargs):
        key = ndb.Key('Project', name)
        project = Project(key=key, **kwargs)
        if save:
            project.put()
        return project