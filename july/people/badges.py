"""
---------------
Julython Badges
---------------

This is where all the logic that drives the awarding of badges.

Badges consist of a counter, metric and a badge meta info. The
badge meta data defines the look of the badge the color, text,
icon and popup text to display.

The counters and badge awards are stored in a large json blob for
each user. When either a new commit for the user is added or the
user profile is displayed the counters are updated. After the counters
are updated the badges are iterated over to see if a new one was
added or if the user completed another badge.

Counters
---------

* Game(year) commit count, the count of the current game.
* Total commit count, the overall number of commits
* Game(year) language set, set of languages in the current game.
* Total language set, set of languages over all games.

Badge Example
-------------

Here is a sample badge::

    class HundredCommits(Badge):
        counter = 'total_commits'
        target = 100
        badge_class = 'fa-trophy expert'
        badge_text = '100+ Commits'
        badge_popup = 'One hundredth Commit'
        show_progress = True

Example badge json blob::

    {
        'total_commits': 1232,
        'total_projects': 34,
        'game_commits': 120,
        'game_days': 20,
        '_current_comment': "current badges are calculated everytime",
        'badges': [
            {
                'title': 'Committed',
                'subtitle': '100+ Commits',
                'count': 200,
                'total': 100,
                'awarded': true,
                'icon': "fa-trophy",
                'level': "novice"
            }
        ],
        '_archived_comment': "This is the list of previous game awards",
        'archived_badges': [
            {
                'title': 'Committed',
                'badge_popup': '100+ Commits in Julython 2012'
                'count': 200,
                'total': 100,
                'awarded': true,
                'icon': "fa-trophy",
                'level': "novice"
            }
        ]
    }

Badge Levels
------------

There are currently 5 levels which are differnent colored icons.

* novice
* journeyman
* expert
* rockstar

"""
import re

from django.core.cache import cache

from july.game.models import Game
from july.people.models import UserBadge, Commit

# TODO(rmyers): copied from django 1.7 remove after we update to it
re_camel_case = re.compile(r'(((?<=[a-z])[A-Z])|([A-Z](?![A-Z]|$)))')


def camel_case_to_dashes(value):
    return re_camel_case.sub(r' \1', value).strip().lower().replace(' ', '_')


class Badge(object):
    """Base badge class"""

    counter = None
    total = 0
    icon = None
    title = ""
    subtitle = ""
    level = ""

    def __init__(self, user_data):
        self.user_data = user_data
        self.count = self.user_data.get(self.counter)

    @property
    def awarded(self):
        return self.count >= self.total

    def to_dict(self):
        return {
            'title': self.title,
            'subtitle': self.subtitle,
            'icon': self.icon,
            'total': self.total,
            'count': self.count,
            'level': self.level,
            'awarded': self.awarded,
        }


class Counter(object):
    """Base Counter Class"""

    query = None
    metric = None

    def __init__(self, user, game=None):
        self.user = user
        self.game = game

    @property
    def name(self):
        return camel_case_to_dashes(self.__class__.__name__)

    @property
    def cache_key(self):
        return '%s-%s' % (self.name, self.user.pk)

    def update(self, user_data):
        "Update the user json with the count from the query"
        cached = cache.get(self.cache_key)
        if cached:
            count_dict = cached
        else:
            count_dict = self.run_query()
            cache.set(self.cache_key, count_dict, timeout=300)
        user_data.update(count_dict)

    def run_query(self):
        """Return the count for this query."""
        q = getattr(self.user, self.query)
        return {self.name: q.count()}


class GameCounter(Counter):
    """Counter for Game Related Counts

    This provides a number of counters for a single game.

    * game_commits (total number of commits in the game)
    * game_days (number of days in the game the user committed)
    """
    metric = 'game'

    def run_query(self):
        if self.game is None:
            self.game = Game.active_or_latest()
        # Commit.calender returns a list of objects for each day a user has
        # commited along with the count during the day. So we can use this
        # query to get the total and the number of days.
        resp = Commit.calendar(self.game, user=self.user)
        objects = resp['objects']
        total = 0
        for obj in objects:
            total += obj.get('commit_count', 0)
        return {
            'game_commits': total,
            'game_days': len(objects)
        }


class TotalCommits(Counter):
    query = 'commit_set'
    metric = 'commits'


class TotalProjects(Counter):
    query = 'projects'
    metric = 'projects'


class FirstCommit(Badge):
    counter = 'total_commits'
    title = 'Welcome Aboard'
    subtitle = 'Thanks for Joining'
    total = 1
    icon = "fa-send"
    level = "novice"


class ThirtyCommits(Badge):
    counter = 'total_commits'
    title = 'A Healthy Start'
    subtitle = '30+ Commits'
    total = 30
    icon = "fa-plus-circle"
    level = "novice"


class HundredCommits(Badge):
    counter = 'total_commits'
    title = 'Highly Committed'
    subtitle = '100+ Commits'
    total = 100
    icon = "fa-plus-circle"
    level = "journeyman"


class FiveHundredCommits(Badge):
    counter = 'total_commits'
    title = 'Outstanding Commitment'
    subtitle = '500+ Commits'
    total = 500
    icon = "fa-plus-circle"
    level = "expert"



class ThousandCommits(Badge):
    counter = 'total_commits'
    title = 'Do You Sleep at All?'
    subtitle = '1000+ Commits'
    total = 1000
    icon = "fa-plus-circle"
    level = "rockstar"


class FiveProjects(Badge):
    counter = 'total_projects'
    title = 'Thanks for Sharing'
    subtitle = '5+ Projects'
    total = 5
    icon = "fa-folder-o"
    level = "novice"


class TenProjects(Badge):
    counter = 'total_projects'
    title = 'Nice Project List'
    subtitle = '10+ Projects'
    total = 10
    icon = "fa-folder-o"
    level = "journeyman"


class FiftyProjects(Badge):
    counter = 'total_projects'
    title = 'Wow just wow'
    subtitle = '50+ Projects'
    total = 50
    icon = "fa-folder-o"
    level = "expert"



class HundredProjects(Badge):
    counter = 'total_projects'
    title = 'You Love Sharing'
    subtitle = '100+ Projects'
    total = 100
    icon = "fa-folder-o"
    level = "rockstar"


BADGES = [
    FirstCommit,
    ThirtyCommits,
    HundredCommits,
    FiveHundredCommits,
    ThousandCommits,
    FiveProjects,
    TenProjects,
    FiftyProjects,
    HundredProjects,
]

COUNTERS = [
    GameCounter,
    TotalCommits,
    TotalProjects,
]


def update_user(user, game=None):
    user_badge, created = UserBadge.objects.get_or_create(user=user)
    user_data = user_badge.badges or {}
    # Update all the counts in user_dict
    for counter in COUNTERS:
        c = counter(user, game=None)
        c.update(user_data)

    user_badges = []
    for badge in BADGES:
        b = badge(user_data)
        user_badges.append(b.to_dict())

    user_data['badges'] = user_badges
    user_badge.badges = user_data
    user_badge.save()
    return user_data
