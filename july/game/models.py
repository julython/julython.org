from collections import namedtuple
import datetime
import logging

from django.conf import settings
from django.db import models
from django.db.models.signals import post_save, m2m_changed
from django.dispatch import receiver
from django.utils import timezone

from july.people.models import Project, Location, Team, Commit, Language


LOCATION_SQL = """\
SELECT july_user.location_id AS slug,
    people_location.name AS name,
    SUM(game_player.points) AS total
    FROM game_player, july_user, people_location
    WHERE game_player.user_id = july_user.id
    AND july_user.location_id = people_location.slug
    AND people_location.approved = 1
    AND game_player.game_id = %s
    GROUP BY july_user.location_id
    ORDER BY total DESC
    LIMIT 50;
"""


TEAM_SQL = """\
SELECT july_user.team_id AS slug,
    people_team.name AS name,
    SUM(game_player.points) AS total
    FROM game_player, july_user, people_team
    WHERE game_player.user_id = july_user.id
    AND july_user.team_id = people_team.slug
    AND people_team.approved = 1
    AND game_player.game_id = %s
    GROUP BY july_user.team_id
    ORDER BY total DESC
    LIMIT 50;
"""


# Number of commits on each day during the game
HISTOGRAM = """\
SELECT count(*), DATE(people_commit.timestamp),
    game_game.start AS start, game_game.end AS end
    FROM people_commit, game_game
    WHERE game_game.id = %s
    AND people_commit.timestamp > start
    AND people_commit.timestamp < end
    GROUP BY DATE(people_commit.timestamp)
    LIMIT 33;
"""


class Game(models.Model):

    start = models.DateTimeField()
    end = models.DateTimeField()
    commit_points = models.IntegerField(default=1)
    project_points = models.IntegerField(default=10)
    problem_points = models.IntegerField(default=5)
    players = models.ManyToManyField(
        settings.AUTH_USER_MODEL, through='Player')
    boards = models.ManyToManyField(Project, through='Board')
    language_boards = models.ManyToManyField(
        Language, through='LanguageBoard')

    class Meta:
        ordering = ['-end']
        get_latest_by = 'end'

    def __unicode__(self):
        if self.end.month == 8:
            return 'Julython %s' % self.end.year
        elif self.end.month == 2:
            return 'J(an)ulython %s' % self.end.year
        else:
            return 'Testathon %s' % self.end.year

    @property
    def locations(self):
        """Preform a raw query to mimic a real model."""
        return Location.objects.raw(LOCATION_SQL, [self.pk])

    @property
    def teams(self):
        """Preform a raw query to mimic a real model."""
        return Team.objects.raw(TEAM_SQL, [self.pk])

    @property
    def histogram(self):
        """Return a histogram of commits during the month"""
        from django.db import connection
        cursor = connection.cursor()
        cursor.execute(HISTOGRAM, [self.pk])
        Day = namedtuple('Day', 'count date start end')

        def mdate(d):
            # SQLITE returns a string while mysql returns date object
            # so make it look the same.
            if isinstance(d, datetime.date):
                return d
            day = datetime.datetime.strptime(d, '%Y-%m-%d')
            return day.date()

        days = {mdate(i.date): i for i in map(Day._make, cursor.fetchall())}
        num_days = self.end - self.start
        records = []
        for day_n in xrange(num_days.days + 1):
            day = self.start + datetime.timedelta(days=day_n)
            records.append(days.get(day.date(), Day(0, day.date(), '', '')))

        logging.debug(records)
        # TODO (rmyers): This should return a json array with labels
        results = [int(day.count) for day in records]
        return results

    @classmethod
    def active(cls, now=None):
        """Returns the active game or None."""
        if now is None:
            now = timezone.now()
        try:
            return cls.objects.get(start__lte=now, end__gte=now)
        except cls.DoesNotExist:
            return None

    @classmethod
    def active_or_latest(cls, now=None):
        """Return the an active game or the latest one."""
        if now is None:
            now = timezone.now()
        game = cls.active(now)
        if game is None:
            query = cls.objects.filter(end__lte=now)
            if len(query):
                game = query[0]
        return game

    def add_points_to_board(self, commit, from_orphan=False):
        board, created = Board.objects.select_for_update().get_or_create(
            game=self, project=commit.project,
            defaults={'points': self.project_points + self.commit_points})
        if not created and not from_orphan:
            board.points += self.commit_points
            board.save()
        return board

    def add_points_to_language_boards(self, commit):
        for language in commit.languages:
            lang, _ = Language.objects.get_or_create(name=language)
            language_board, created = LanguageBoard.objects. \
                select_for_update().get_or_create(
                    game=self, language=lang,
                    defaults={'points': self.commit_points})
            if not created:
                language_board.points += self.commit_points
                language_board.save()

    def add_points_to_player(self, board, commit):
        player, created = Player.objects.select_for_update().get_or_create(
            game=self, user=commit.user,
            defaults={'points': self.project_points + self.commit_points})
        player.boards.add(board)
        if not created:
            # we need to get the total points for the user
            project_points = player.boards.all().count() * self.project_points
            commit_points = Commit.objects.filter(
                user=commit.user,
                timestamp__gte=self.start,
                timestamp__lte=self.end).count() * self.commit_points
            # TODO (rmyers): Add in problem points
            player.points = project_points + commit_points
            player.save()

    def add_commit(self, commit, from_orphan=False):
        """
        Add a commit to the game, update the scores for the player/boards.
        If the commit was previously an orphan commit don't update the board
        total, since it was already updated.

        TODO (rmyers): This may need to be run by celery in the future instead
        of a post create signal.
        """
        board = self.add_points_to_board(commit, from_orphan)
        self.add_points_to_language_boards(commit)

        if commit.user:
            self.add_points_to_player(board, commit)


class Player(models.Model):
    """A player in the game."""

    game = models.ForeignKey(Game)
    user = models.ForeignKey(settings.AUTH_USER_MODEL)
    points = models.IntegerField(default=0)
    boards = models.ManyToManyField('Board')

    class Meta:
        ordering = ['-points']
        get_latest_by = 'game__end'

    def __unicode__(self):
        return unicode(self.user)


class AbstractBoard(models.Model):
    """Keeps points per metric per game"""
    game = models.ForeignKey(Game)
    points = models.IntegerField(default=0)

    class Meta:
        abstract = True
        ordering = ['-points']
        get_latest_by = 'game__end'


class Board(AbstractBoard):
    """A project with commits in the game."""

    project = models.ForeignKey(Project)

    def __unicode__(self):
        return 'Board for %s' % unicode(self.project)


class LanguageBoard(AbstractBoard):
    """A language with commits in the game."""

    language = models.ForeignKey(Language)

    def __unicode__(self):
        return 'Board for %s' % unicode(self.language)


@receiver(post_save, sender=Commit)
def add_commit(sender, **kwargs):
    """Listens for new commits and adds them to the game."""
    commit = kwargs.get('instance')
    active_game = Game.active(now=commit.timestamp)
    if active_game is not None:
        from_orphan = not kwargs.get('created', False)
        active_game.add_commit(commit, from_orphan=from_orphan)
