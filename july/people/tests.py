
import datetime
import json
from pytz import UTC

from django.test.client import Client
from django.test import TestCase
from django.conf import settings
from django.template.defaultfilters import slugify

from july.models import User
from july.people.models import Location, Project, Commit, Team
from july.game.models import Game, Board, Player


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
    START = datetime.datetime(year=2012, month=12, day=1, tzinfo=UTC)
    END = datetime.datetime(year=2012, month=12, day=21, tzinfo=UTC) # End of time itself
    
    def setUp(self):
        self.user = self.make_user(self.USER)
        self.user.add_auth_id(self.AUTH_ID)
        self.game = Game.objects.create(start=self.START, end=self.END)
    
    def make_user(self, username, **kwargs):
        return User.objects.create_user(username=username, **kwargs)
    
    def make_location(self, location):
        slug = slugify(location)
        return Location.objects.create(name=location, slug=slug)
    
    def make_team(self, team):
        slug = slugify(team)
        return Team.objects.create(name=team, slug=slug)

    def test_post_creates_commits(self):
        resp = self.client.post(self.API_URL, self.POST)
        resp_body = json.loads(resp.content)
        self.assertEqual(resp.status_code, 201)
        self.assertEqual(len(resp_body['commits']), 2)
    
    def test_post_adds_points_to_user(self):
        self.client.post(self.API_URL, self.POST)
        u = Player.objects.get(game=self.game, user=self.user)
        self.assertEqual(u.points, 12)
    
    def test_post_adds_points_to_project(self):
        self.client.post(self.API_URL, self.POST)
        p = Board.objects.get(game=self.game, project__slug=self.PROJECT_SLUG)
        self.assertEqual(p.points, 12)
    
    def test_post_adds_project_to_commit(self):
        resp = self.client.post(self.API_URL, self.POST)
        resp_body = json.loads(resp.content)
        c_hash = resp_body['commits'][0]
        commit = Commit.objects.get(hash=c_hash)
        self.assertEqual(commit.project.url, self.PROJECT_URL)
        self.assertEqual(commit.project.slug, self.PROJECT_SLUG)
    
    def test_post_adds_points_to_location(self):
        location = self.make_location('Austin, TX')
        self.user.location = location
        self.user.save()
        self.client.post(self.API_URL, self.POST)
        self.assertEqual(self.game.locations[0].total, 12)

    def test_post_adds_points_to_team(self):
        team = self.make_team('Rackers')
        self.user.team = team
        self.user.save()
        self.client.post(self.API_URL, self.POST)
        self.assertEqual(self.game.teams[0].total, 12)


class GithubTest(SCMTestMixin, TestCase):
    
    USER = 'defunkt'
    AUTH_ID = 'email:chris@ozmm.org'
    API_URL = '/api/v1/github'
    PROJECT_URL = 'http://github.com/defunkt/github'
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
                "timestamp": "2012-12-15T14:57:17-08:00",
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
                "timestamp": "2012-12-15T14:36:34-08:00"
              }
            ],
            "after": "de8251ff97ee194a289832576287d6f8ad74e3d0",
            "ref": "refs/heads/master"
        })
    }
    
class BitbucketHandlerTests(SCMTestMixin, TestCase):
    
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
                    "timestamp": "2012-12-05 06:07:03", 
                    "utctimestamp": "2012-12-05 04:07:03+00:00"
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
                    "timestamp": "2012-12-06 06:07:03", 
                    "utctimestamp": "2012-12-06 04:07:03+00:00"
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