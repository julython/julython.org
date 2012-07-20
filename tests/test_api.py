
import unittest
import json
import logging
import datetime
import base64

from google.appengine.ext import ndb, deferred

from utils import WebTestCase

from gae_django.auth.models import User
from july.api import app, make_digest, utcdatetime
from july.people.models import Project, Location, Accumulator
from july import settings
from july.live.models import Message

class CommitApiTests(WebTestCase):
    
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
    
    def test_error_response(self):
        self.make_user('josh', email='josh@example.com')
        # post missing required field
        body = { "name": "Josh Marshall"}
        body_json = json.dumps(body)
        resp = self.app.post('/api/v1/commits', body_json, headers={"Authorization": make_digest('josh')}, status=400)
        resp_body = json.loads(resp.body)
        self.assertEqual(resp_body['message'], ['This field is required.'])
    
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
        resp = self.app.get('/api/v1/commits/%s' % resp_body['commits'][0])
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
        resp = self.app.post('/api/v1/commits', body_json, 
            headers={"Authorization": make_digest('josh')})
        resp_body = json.loads(resp.body)
        
        # try to get the commit from the api
        resp = self.app.get('/api/v1/commits/%s' % resp_body['commits'][0])
        commit = json.loads(resp.body)
        self.assertEqual(commit['name'], "Josh Marshall")
        self.assertEqual(commit['message'], "Working on Tornado stuff!")
        self.assertEqual(commit['url'], "https://github.com/project/commitID")
        self.assertEqual(commit['timestamp'], 5430604985)
        self.assertEqual(commit['hash'], "6a87af2a7eb3de1e17ac1cce41e060516b38c0e9")
    
    def test_commit_not_found(self):
        resp = self.app.get('/api/v1/commits/blah', status=404)
        self.assertEqual(resp.status_code, 404)
    
    def test_filter_by_user(self):
        user = self.make_user(username="josh")
        user2 = self.make_user(username="sam")
        self.make_commit(user, message='Working on the pythons')
        self.make_commit(user, message="Still working on the pythons")
        self.make_commit(user2, message="Sam working on the pythons")
        
        resp = self.app.get('/api/v1/commits?filter=josh')
        resp_body = json.loads(resp.body)
        
        self.assertEqual(len(resp_body['models']), 2)
    
    def test_filter_by_user_with_at_symbol(self):
        user = self.make_user(username="twitter:josh")
        user2 = self.make_user(username="sam")
        self.make_commit(user, message='Working on the pythons')
        self.make_commit(user, message="Still working on the pythons")
        self.make_commit(user2, message="Sam working on the pythons")
        
        resp = self.app.get('/api/v1/commits?filter=@josh')
        resp_body = json.loads(resp.body)
        
        self.assertEqual(len(resp_body['models']), 2)
    
    def test_options(self):
        resp = self.app.options('/api/v1/commits')
        self.assertEqual(resp.status_code, 200)
        self.assertEqual(resp.headers.get('Allow'), 'GET, POST, OPTIONS')

class ProjectApiTests(WebTestCase):
    
    APPLICATION = app
    
    def test_get_project(self):
        self.make_project("fancypants", url='http://blah.com')
        resp = self.app.get("/api/v1/projects/fancypants")
        resp_body = json.loads(resp.body)
        self.assertEqual(resp_body['url'], 'http://blah.com')
    
    def test_not_found(self):
        resp = self.app.get('/api/v1/project/fancypants', status=404)
        self.assertEqual(resp.status_code, 404)

    def test_create_existing_returns_200(self):
        self.make_user('josh', email='josh@example.com')
        self.make_project("gh-user-project", url='http://github.com/user/project')
        body_json = json.dumps({'url': 'http://github.com/user/project'})
        resp = self.app.post('/api/v1/projects', body_json, 
            headers={"Authorization": make_digest('josh')}
        )
        self.assertEqual(resp.status_code, 200)
    
    def test_create_new_project(self):
        self.make_user('josh', email='josh@example.com')
        body_json = json.dumps({'url': 'http://github.com/user/project'})
        resp = self.app.post('/api/v1/projects', body_json, 
            headers={"Authorization": make_digest('josh')}
        )
        self.assertEqual(resp.status_code, 201)
    
    def test_malformed_post(self):
        self.make_user('josh', email='josh@example.com')
        body_json = json.dumps({'bad': 'http://github.com/user/project'})
        resp = self.app.post('/api/v1/projects', body_json, 
            headers={"Authorization": make_digest('josh')},
        status=400)
        resp_body = json.loads(resp.body)
        self.assertEqual(resp_body['url'], ["This field is required."])
    
    def test_user_get(self):
        self.make_user('twitter:josh', email='josh@example.com')
        resp = self.app.get('/api/v1/people/josh')
        self.assertEqual(resp.status_code, 200)
    
    def test_user_dynamic_properties(self):
        self.make_user('twitter:josh', foo_tastic='super-property')
        resp = self.app.get('/api/v1/people/josh')
        person = json.loads(resp.body)
        self.assertEqual(person['foo_tastic'], 'super-property')

    def test_options(self):
        resp = self.app.options('/api/v1/people')
        self.assertEqual(resp.status_code, 200)
        self.assertEqual(resp.headers.get('Allow'), 'GET, OPTIONS')

class GithubHandlerTests(WebTestCase):
    
    APPLICATION = app
    
    PAYLOAD = {
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
    }
    
    POST = {'payload':
        json.dumps(PAYLOAD)
    }
    
    def test_post_creates_commits(self):
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        resp = self.app.post('/api/v1/github', self.POST)
        resp_body = json.loads(resp.body)
        self.assertEqual(len(resp_body['commits']), 2)
    
    def test_post_adds_points_to_user(self):
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        self.app.post('/api/v1/github', self.POST)
        u = User.get_by_auth_id('email:chris@ozmm.org')
        self.assertEqual(u.total, 12)
    
    def test_post_adds_points_to_project(self):
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        self.app.post('/api/v1/github', self.POST)
        p_key = Project.make_key('http://github.com/defunkt/github')
        p = p_key.get()
        self.assertEqual(p.total, 12)
    
    def test_post_adds_project_slug_to_commit(self):
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        resp = self.app.post('/api/v1/github', self.POST)
        resp_body = json.loads(resp.body)
        commit_key = ndb.Key(urlsafe=resp_body['commits'][0])
        commit = commit_key.get()
        self.assertEqual(commit.project_slug, 'gh-defunkt-github')
        
    def test_post_adds_project_to_commit(self):
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        resp = self.app.post('/api/v1/github', self.POST)
        resp_body = json.loads(resp.body)
        commit_key = ndb.Key(urlsafe=resp_body['commits'][0])
        commit = commit_key.get()
        self.assertEqual(commit.project, 'http://github.com/defunkt/github')
    
    def test_post_adds_points_to_location(self):
        self.policy.SetProbability(1)
        user = self.make_user('chris', location='Austin TX')
        user.add_auth_id('email:chris@ozmm.org')
        self.app.post('/api/v1/github', self.POST)
        tasks = self.taskqueue_stub.GetTasks('default')
        self.assertEqual(5, len(tasks))

        # Run the task
        task = tasks[0]
        deferred.run(base64.b64decode(task['body']))
    
        location = Location.get_by_id('austin-tx')
        self.assertEqual(location.total, 12)
        
    def test_post_adds_points_to_global(self):
        self.policy.SetProbability(1)
        user = self.make_user('chris', location='Austin TX')
        user.add_auth_id('email:chris@ozmm.org')
        self.app.post('/api/v1/github', self.POST)
        tasks = self.taskqueue_stub.GetTasks('default')
        self.assertEqual(5, len(tasks))

        # Run the task
        for task in tasks:
            deferred.run(base64.b64decode(task['body']))
    
        stats = Accumulator.get_histogram('global')
        self.assertListEqual(stats, [
            0,0,0,0,0,0,0,0,0,0,
            0,0,0,0,2,0,0,0,0,0,
            0,0,0,0,0,0,0,0,0,0,0,
        ])
                           
    def test_testing_mode_off(self):
        settings.TESTING = False
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        self.app.post('/api/v1/github', self.POST)
        p_key = Project.make_key('http://github.com/defunkt/github')
        # project should not be created
        self.assertEqual(p_key.get(), None)
    
    def test_testing_mode_off_user_points(self):
        settings.TESTING = False
        user = self.make_user('chris')
        user.add_auth_id('email:chris@ozmm.org')
        self.app.post('/api/v1/github', self.POST)
        u = User.get_by_auth_id('email:chris@ozmm.org')
        total = getattr(u, 'total', None)
        self.assertEqual(total, None)


class BitbucketHandlerTests(WebTestCase):
    
    APPLICATION = app
    
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
        resp = self.app.post('/api/v1/bitbucket', self.POST)
        resp_body = json.loads(resp.body)
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
        self.policy.SetProbability(1)
        user = self.make_user('marcus', location='Austin TX')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        tasks = self.taskqueue_stub.GetTasks('default')
        self.assertEqual(3, len(tasks))

        # Run the task
        task = tasks[0]
        deferred.run(base64.b64decode(task['body']))
    
        location = Location.get_by_id('austin-tx')
        self.assertEqual(location.total, 11)       
    
    def test_post_adds_points_to_global(self):
        self.policy.SetProbability(1)
        user = self.make_user('marcus', location='Austin TX')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        tasks = self.taskqueue_stub.GetTasks('default')
        self.assertEqual(3, len(tasks))

        # Run the task
        for task in tasks:
            deferred.run(base64.b64decode(task['body']))
    
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

class BitbucketLikeHandlerTests(WebTestCase):
    
    APPLICATION = app
    
    POST = {'payload':
        json.dumps({
            "canon_url": "https://some.other.org", 
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
        resp = self.app.post('/api/v1/bitbucket', self.POST)
        resp_body = json.loads(resp.body)
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
        p_key = Project.make_key('https://some.other.org/marcus/project-x/')
        p = p_key.get()
        self.assertEqual(p.total, 11)
    
    def test_project_url(self):
        user = self.make_user('marcus')
        user.add_auth_id('email:marcus@somedomain.com')
        self.app.post('/api/v1/bitbucket', self.POST)
        user = User.get_by_auth_id('email:marcus@somedomain.com')
        self.assertTrue('https://some.other.org/marcus/project-x/' in user.projects)

class TestUtils(unittest.TestCase):
    
    def test_utcdatetime_removes_tzinfo(self):
        ts = '2012-05-30 04:07:03+00:00'
        dt = utcdatetime(ts)
        self.assertEqual(dt.tzinfo, None)
    
    def test_utcdatetime_offset(self):
        ts = '2012-05-30 04:07:03-06:00'
        dt = utcdatetime(ts)
        self.assertEqual(dt.hour, 10)

class StatsTest(WebTestCase):
    
    APPLICATION = app
        
    def test_user_counts(self):
        resp = self.app.get('/api/v1/stats/commits/own:fred')
        resp_data = json.loads(resp.body)
        self.assertEqual(resp_data, {
            u"stats": [
                0,0,0,0,0,0,0,0,0,0,
                0,0,0,0,0,0,0,0,0,0,
                0,0,0,0,0,0,0,0,0,0,0
            ],
            u'total': 0,
            u"metric": u"own:fred",
        })
    
    def test_global_counts(self):
        resp = self.app.get('/api/v1/stats/commits/global')
        resp_data = json.loads(resp.body)
        self.assertEqual(resp_data, {
            u"stats": [
                0,0,0,0,0,0,0,0,0,0,
                0,0,0,0,0,0,0,0,0,0,
                0,0,0,0,0,0,0,0,0,0,0
            ],
            u'total': 0,
            u"metric": u"global",
        })

class TestMessage(WebTestCase):
    
    APPLICATION = app
    
    DATA = {
        "username": "github",
        "message": "Something interesting",
        "picture_url": "http://avator.com/my_image.jpg",
        "url": "http://github.com/defunkt/github/commit/k20lkjs920lskjlskjjd1",
        "commit_hash": "k20lkjs920lskjlskjjd1",
        "project": "http://github.com/defunkt/github"}
    
    def test_create_message(self):
        data = json.dumps(self.DATA)
        self.app.post('/api/v1/live', data)
        count = Message.query().count(10)
        self.assertEqual(count, 1)
    
    def test_post_adds_points_to_global(self):
        data = json.dumps(self.DATA)
        self.app.post('/api/v1/live', data)
        tasks = self.taskqueue_stub.GetTasks('live')
        self.assertEqual(1, len(tasks))

        # Run the task
        for task in tasks:
            deferred.run(base64.b64decode(task['body']))
        
        #TODO: assert channels 

if __name__ == '__main__':
    logging.getLogger().setLevel(logging.DEBUG)
    unittest.main()
      
    
