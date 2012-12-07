
import json

from django.test.client import Client
from django.test import TestCase
from django.conf import settings
from django.template.defaultfilters import slugify

from july.models import User
from july.people.models import Location, Project, Commit, Team


class ApiTestCase(TestCase):
    
    def make_user(self, username, **kwargs):
        return User.objects.create_user(username=username, **kwargs)
    
    def make_location(self, location):
        slug = slugify(location)
        return Location.objects.create(name=location, slug=slug)
    
    def setUp(self):
        self.app = Client()


class BitbucketHandlerTests(ApiTestCase):
    
    POST = {'payload':
        json.dumps({
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
                "fork": False, 
                "is_private": True, 
                "name": "Project X", 
                "owner": "marcus", 
                "scm": "hg", 
                "slug": "project-x", 
                "website": ""
            }, 
            "user": "marcus"
        })
    }
    
    def test_post_creates_commits(self):
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        resp = self.client.post('/api/v1/bitbucket/', self.POST)
        resp_body = json.loads(resp.content)
        self.assertEqual(resp.status_code, 201)
        self.assertEqual(len(resp_body['commits']), 1)
    
    def test_post_adds_points_to_user(self):
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        u = User.get_by_auth_id('email:marcus@somedomain.com')
        self.assertEqual(u.total, 11)
    
    def test_post_adds_points_to_project(self):
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        p_key = Project.make_key('https://bitbucket.org/marcus/project-x/')
        p = p_key.get()
        self.assertEqual(p.total, 11)
    
    def test_post_adds_project_slug_to_commit(self):
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        resp = self.app.post('/api/v1/bitbucket', self.POST)
        resp_body = json.loads(resp.body)
        commit_key = ndb.Key(urlsafe=resp_body['commits'][0])
        commit = commit_key.get()
        self.assertEqual(commit.project_slug, 'bb-marcus-project-x')
        
    def test_post_adds_project_to_commit(self):
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        resp = self.app.post('/api/v1/bitbucket', self.POST)
        resp_body = json.loads(resp.body)
        commit_key = ndb.Key(urlsafe=resp_body['commits'][0])
        commit = commit_key.get()
        self.assertEqual(commit.project, 'https://bitbucket.org/marcus/project-x/')
    
    def test_post_adds_points_to_location(self):
        location = self.make_location('Austin, TX')
        user = self.make_user('marcus', location=location)
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
    
        self.assertEqual(location.total, 11)       
    
    def test_post_adds_points_to_global(self):
        location = self.make_location('Austin, TX')
        user = self.make_user('marcus', location=location)
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
    
        stats = Accumulator.get_histogram('global')
        self.assertListEqual(stats, [
            0,0,0,0,0,0,0,0,0,0,
            0,0,0,0,0,0,0,0,0,0,
            0,0,0,0,0,0,0,0,0,1,0,
        ])
    
    def test_testing_mode_off(self):
        settings.TESTING = False
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        u = User.get_by_auth_id('email:marcus@somedomain.com')
        p_key = Project.make_key('https://bitbucket.org/marcus/project-x/')
        # project should not be created
        self.assertEqual(p_key.get(), None)
    
    def test_testing_mode_off_user_points(self):
        settings.TESTING = False
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        u = User.get_by_auth_id('email:marcus@somedomain.com')
        total = getattr(u, 'total', None)
        self.assertEqual(total, None)