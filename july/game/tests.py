
import datetime

import uuid
from pytz import UTC

from django.test import TestCase
from django.template.defaultfilters import slugify
from django.utils import timezone

from july.models import User
from july.people.models import Location, Commit, Team, Project
from july.game.models import Game
from july.game.views import GameMixin


class ModelMixin(object):

    def make_game(self, start=None, end=None):
        now = timezone.now()
        if start is None:
            start = now - datetime.timedelta(days=1)
        if end is None:
            end = now + datetime.timedelta(days=1)
        return Game.objects.create(start=start, end=end)

    def make_user(self, username, **kwargs):
        return User.objects.create_user(username=username, **kwargs)

    def make_location(self, location):
        slug = slugify(location)
        return Location.objects.create(name=location, slug=slug)

    def make_team(self, team):
        slug = slugify(team)
        return Team.objects.create(name=team, slug=slug)

    def make_project(self, url='http://github.com/project', name='test'):
        return Project.create(url=url, name=name)

    def make_commit(self, auth_id='x:no', hash=None, timestamp=None,
                    project=None, **kwargs):
        if hash is None:
            hash = str(uuid.uuid4())
        if timestamp is None:
            timestamp = timezone.now()
        commit = kwargs.copy()
        commit.update({'hash': hash, 'timestamp': timestamp})
        return Commit.create_by_auth_id(auth_id, [commit], project=project)


class GameModelTests(TestCase, ModelMixin):

    def setUp(self):
        self.now = timezone.now()
        self.yesterday = self.now - datetime.timedelta(days=1)
        self.tomorrow = self.now + datetime.timedelta(days=1)
        self.early = self.now - datetime.timedelta(days=2)
        self.late = self.now + datetime.timedelta(days=2)

    def test_julython(self):
        game = self.make_game(
            end=datetime.datetime(year=2012, month=8, day=2, tzinfo=UTC))
        self.assertEqual(unicode(game), 'Julython 2012')

    def test_janulython(self):
        game = self.make_game(
            end=datetime.datetime(year=2012, month=2, day=2, tzinfo=UTC))
        self.assertEqual(unicode(game), 'J(an)ulython 2012')

    def test_testathon(self):
        game = self.make_game(
            end=datetime.datetime(year=2012, month=5, day=2, tzinfo=UTC))
        self.assertEqual(unicode(game), 'Testathon 2012')

    def test_active(self):
        game = self.make_game()
        active = Game.active()
        self.assertEqual(active, game)

    def test_active_or_latest(self):
        game = self.make_game()
        active = Game.active_or_latest()
        self.assertEqual(active, game)

    def test_active_or_latest_future(self):
        self.make_game(start=self.tomorrow, end=self.late)
        active = Game.active_or_latest()
        self.assertEqual(active, None)

    def test_active_or_latest_past(self):
        game = self.make_game(start=self.early, end=self.yesterday)
        active = Game.active_or_latest()
        self.assertEqual(active, game)

    def test_not_active(self):
        self.make_game(start=self.tomorrow, end=self.late)
        active = Game.active()
        self.assertEqual(active, None)

    def test_add_board(self):
        # Test the post add hook
        game = self.make_game()
        project = self.make_project()
        self.make_commit(project=project)
        self.assertEqual(len(game.boards.all()), 1)
        self.assertEqual(unicode(game.boards.get()), 'test')

    def test_add_player(self):
        game = self.make_game()
        project = self.make_project()
        user = self.make_user('ted')
        user.add_auth_id('test:ted')
        self.make_commit(auth_id='test:ted', project=project)
        self.assertEqual(len(game.players.all()), 1)
        self.assertEqual(unicode(game.players.get()), 'ted')

    def test_histogram(self):
        game = self.make_game()
        project = self.make_project()
        self.make_commit(project=project)
        self.make_commit(project=project)
        self.make_commit(project=project)
        self.make_commit(project=project)
        self.assertEqual(game.histogram, [0, 4, 0])

    def test_histogram_end(self):
        # TODO: Fix histogram
        delta = datetime.timedelta(days=31)
        game = self.make_game(start=self.now - delta, end=self.now)
        project = self.make_project()
        self.make_commit(project=project, timestamp=self.tomorrow - delta)
        self.make_commit(project=project, timestamp=self.yesterday)
        self.make_commit(project=project, timestamp=self.yesterday)
        self.make_commit(project=project, timestamp=self.yesterday)
        self.assertEqual(game.histogram, [0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                                          0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                                          0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0])


class Mixer(GameMixin):
    """Helper class to test mixin"""
    def __init__(self, **kwargs):
        self.kwargs = kwargs


class GameViewTests(TestCase, ModelMixin):

    def setUp(self):
        self.now = timezone.now()
        self.yesterday = self.now - datetime.timedelta(days=1)
        self.tomorrow = self.now + datetime.timedelta(days=1)
        self.early = self.now - datetime.timedelta(days=2)
        self.late = self.now + datetime.timedelta(days=2)

    def test_game_mixin_active(self):
        active = self.make_game()
        mixed = Mixer()
        game = mixed.get_game()
        self.assertEqual(game, active)

    def test_game_mixin_latest(self):
        past = self.make_game(start=self.early, end=self.yesterday)
        mixed = Mixer()
        game = mixed.get_game()
        self.assertEqual(game, past)

    def test_game_mixin_future(self):
        past = self.make_game(start=self.early, end=self.yesterday)
        future = self.make_game(start=self.tomorrow, end=self.late)
        mixed = Mixer()
        game = mixed.get_game()
        self.assertNotEqual(game, future)
        self.assertEqual(game, past)

    def test_game_mixin_old(self):
        recent = self.make_game(start=self.early, end=self.yesterday)
        start = self.early - datetime.timedelta(days=4)
        middle = self.early - datetime.timedelta(days=3)
        end = self.early - datetime.timedelta(days=2)
        past = self.make_game(start=start, end=end)
        mixed = Mixer(year=middle.year, month=middle.month, day=middle.day)
        game = mixed.get_game()
        self.assertNotEqual(game, recent)
        self.assertEqual(game, past)

    def test_player_view(self):
        self.make_game()
        project = self.make_project()
        user = self.make_user('ted')
        user.add_auth_id('test:ted')
        self.make_commit(auth_id='test:ted', project=project)
        resp = self.client.get('/people/')
        self.assertContains(resp, 'ted')

    def test_project_view(self):
        self.make_game()
        project = self.make_project(name="fred")
        user = self.make_user('ted')
        user.add_auth_id('test:ted')
        self.make_commit(auth_id='test:ted', project=project)
        resp = self.client.get('/projects/')
        self.assertContains(resp, "fred")

    def test_loction_view(self):
        self.make_game()
        project = self.make_project()
        location = self.make_location("Austin, TX")
        user = self.make_user('ted', location=location)
        user.add_auth_id('test:ted')
        self.make_commit(auth_id='test:ted', project=project)
        resp = self.client.get('/location/')
        self.assertContains(resp, "Austin, TX")

    def test_team_view(self):
        self.make_game()
        project = self.make_project()
        team = self.make_team("Commit Rangers")
        user = self.make_user('ted', team=team)
        user.add_auth_id('test:ted')
        self.make_commit(auth_id='test:ted', project=project)
        resp = self.client.get('/teams/')
        self.assertContains(resp, "Commit Rangers")

    def test_event_handler(self):
        resp = self.client.post('/events/pub/test/', {"foo": "bar"})
        self.assertEqual(resp.status_code, 200)
