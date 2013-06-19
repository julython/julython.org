
from collections import namedtuple

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
    players = models.ManyToManyField(settings.AUTH_USER_MODEL, through='Player')
    boards = models.ManyToManyField(Project, through='Board')
    language_boards = models.ManyToManyField(Language,
        through='LanguageBoard')

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
        days = map(Day._make, cursor.fetchall())
        # TODO (rmyers): This should be moved to view or templatetag?
        # return just the totals for now and condense and pad the results
        # so that there are 31 days. The games start noon UTC time the last
        # day of the previous month and end noon the 1st of the next month.
        # This, is, really, ugly, don't look!
        results = [int(day.count) for day in days]
        if len(results) >= 2:
            results[1] += results[0]
            results = results[1:]  # trim the first day
        if len(results) == 32:
            results[30] += results[31]
            results = results[:31]  # trim the last day
        padding = [0 for day in xrange(31 - len(results))]
        results += padding
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
    def active_or_latest(cls):
        """Return the an active game or the latest one."""
        game = cls.active()
        if game is None:
            game = cls.objects.latest()
        return game

    def add_points_to_board(self, commit, from_orphan=False):
        board, created = Board.objects.select_for_update().get_or_create(
            game=self, project=commit.project,
            defaults={'points': self.project_points + self.commit_points})
        if not created and not from_orphan:
            board.points += self.commit_points
            board.save()
        return board

    def add_points_to_language_boards(self, languages, user):
        language_boards = []
        for language in languages:
            language_board, created = LanguageBoard.objects.select_for_update().get_or_create(
                game=self, language=language,
                defaults={'points': self.commit_points})
            if not created:
                language_board.points += self.commit_points
                language_board.save()
            language_boards.append(language_board)
        if user:
            player = Player.objects.get(game=self, user=user)
            player.language_boards.add(*language_boards)

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
        if commit.user:
            self.add_points_to_player(board, commit)


class Player(models.Model):
    """A player in the game."""

    game = models.ForeignKey(Game)
    user = models.ForeignKey(settings.AUTH_USER_MODEL)
    points = models.IntegerField(default=0)
    boards = models.ManyToManyField('Board')
    language_boards = models.ManyToManyField('LanguageBoard')

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


@receiver(m2m_changed, sender=Commit.languages.through)
def add_points_to_language_boards(sender, **kwargs):
    """
    Listens to languages being added to commits, adds point to language boards
    """
    if not kwargs['action'] == 'post_add':
        return
    commit = kwargs.get('instance')
    active_game = Game.active(now=commit.timestamp)
    if active_game is not None:
        user = commit.user
        languages_added = Language.objects.filter(name__in=list(kwargs['pk_set']))
        active_game.add_points_to_language_boards(languages_added, user)
