
import json

from django.test.client import Client
from django.test import TestCase
from django.conf import settings
from django.template.defaultfilters import slugify

from july.models import User
from july.people.models import Location, Project, Commit, Team


class SCMTestMixin(object):
    """
    All scm endpoints should behave the same, and thus this set of test cases
    shall prove that statment. 
    """
    
    AUTH_ID = "email:ted@example.com"
    API_URL = ""
    PROJECT_URL = ""
    USER = "bobby"
    POST = ""
    
    def make_user(self, username, **kwargs):
        return User.objects.create_user(username=username, **kwargs)
    
    def make_location(self, location):
        slug = slugify(location)
        return Location.objects.create(name=location, slug=slug)

    def test_post_creates_commits(self):
        user = self.make_user(self.USER)
        user.add_auth_id(self.AUTH_ID)
        resp = self.client.post(self.API_URL, self.POST)
        resp_body = json.loads(resp.content)
        self.assertEqual(resp.status_code, 201)
        self.assertEqual(len(resp_body['commits']), 2)
    
    def test_post_adds_points_to_user(self):
        user = self.make_user(self.USER)
        user.add_auth_id(self.AUTH_ID)
        self.client.post(self.API_URL, self.POST)
        u = User.get_by_auth_id(self.AUTH_ID)
        self.assertEqual(u.total, 12)
    
    def test_post_adds_points_to_project(self):
        user = self.make_user(self.USER)
        user.add_auth_id(self.AUTH_ID)
        self.client.post(self.API_URL, self.POST)
        p = Project.objects.get(slug=self.PROJECT_URL)
        self.assertEqual(p.total, 12)
    
    def test_post_adds_project_to_commit(self):
        user = self.make_user(self.USER)
        user.add_auth_id(self.AUTH_ID)
        resp = self.client.post(self.API_URL, self.POST)
        resp_body = json.loads(resp.content)
        c_hash = resp_body['commits'][0]
        commit = Commit.objects.get(hash=c_hash)
        self.assertEqual(commit.project.slug, self.PROJECT_SLUG)
        self.assertEqual(commit.project.url, self.PROJECT_URL)
    
    def test_post_adds_points_to_location(self):
        location = self.make_location('Austin, TX')
        user = self.make_user('marcus', location=location)
        user.add_auth_id(self.AUTH_ID)
        self.client.post(self.API_URL, self.POST)
        self.assertEqual(location.total, 12)       


class GithubTest(TestCase, SCMTestMixin):
    
    USER = 'defunkt'
    AUTH_ID = 'email:chris@ozmm.org'
    API_URL = '/api/v1/github'
    PROJECT_URL = 'https://github.com/defunkt/github'
    PROJECT_SLUG = 'gh-defunkt-github'
    POST = {'payload':
        json.dumps({
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
        })
    }
    
class BitbucketHandlerTests(TestCase, SCMTestMixin):
    
    USER = 'marcus'
    AUTH_ID = 'email:marcus@somedomain.com'
    API_URL = '/api/v1/bitbucket'
    PROJECT_URL = 'https://bitbucket.org/marcus/project-x/'
    PROJECT_SLUG = 'bb-marcus-project-x'
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
                    "raw_node": "1c0cd3b6f339bb95bfed14d26a93fd28d3166fa8", 
                    "revision": 3, 
                    "size": -1, 
                    "timestamp": "2012-05-30 06:07:03", 
                    "utctimestamp": "2012-05-30 04:07:03+00:00"
                },
                {
                    "author": "marcus", 
                    "branch": "featureB", 
                    "files": [
                        {
                            "file": "somefile.py", 
                            "type": "modified"
                        }
                    ], 
                    "message": "Added some featureB things", 
                    "node": "d14d26a93fd2", 
                    "parents": [
                        "1b458191f31a"
                    ], 
                    "raw_author": "Marcus Bertrand <marcus@somedomain.com>", 
                    "raw_node": "d14d26a93fd28d3166fa81c0cd3b6f339bb95bfe", 
                    "revision": 3, 
                    "size": -1, 
                    "timestamp": "2012-05-29 06:07:03", 
                    "utctimestamp": "2012-05-29 04:07:03+00:00"
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