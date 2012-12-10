
import datetime

from django.conf import settings
from django.db import models
from django.db.models.signals import post_save
from django.dispatch import receiver

from july.people.models import Project, Location, Team, Commit


LOCATION_SQL = """\
SELECT july_user.location_id AS slug,
    people_location.name AS name,
    sum(game_player.points) AS total 
    FROM game_player 
    LEFT JOIN july_user
    LEFT JOIN people_location 
    WHERE game_player.user_id = july_user.id
    AND july_user.location_id = people_location.slug 
    AND game_player.game_id = %s
    GROUP BY july_user.location_id 
    ORDER BY points DESC;
"""


TEAM_SQL = """\
SELECT july_user.team_id AS slug,
    people_team.name AS name,
    sum(game_player.points) AS total 
    FROM game_player 
    LEFT JOIN july_user
    LEFT JOIN people_team 
    WHERE game_player.user_id = july_user.id
    AND july_user.team_id = people_team.slug 
    AND game_player.game_id = %s
    GROUP BY july_user.team_id 
    ORDER BY points DESC;
"""


class Game(models.Model):
    
    start = models.DateTimeField()
    end = models.DateTimeField()
    commit_points = models.IntegerField(default=1)
    project_points = models.IntegerField(default=10)
    problem_points = models.IntegerField(default=5)
    players = models.ManyToManyField(settings.AUTH_USER_MODEL, through='Player')
    boards = models.ManyToManyField(Project, through='Board')
    
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
    
    @classmethod
    def active(cls, now=None):
        """Returns the active game or None."""
        if now is None:
            now = datetime.datetime.now()
        try:
            return cls.objects.get(start__lte=now, end__gte=now)
        except cls.DoesNotExist:
            return None
    
    def add_commit(self, commit, from_orphan=False):
        """
        Add a commit to the game, update the scores for the player/board.
        If the commit was previously an orphan commit don't update the board
        total, since it was already updated.
        
        TODO (rmyers): This may need to be run by celery in the future instead 
        of a post create signal.
        """
        board, created = Board.objects.select_for_update().get_or_create(
            game=self, project=commit.project, 
            defaults={'points': self.project_points + self.commit_points})
        if not created and not from_orphan:
            board.points += self.commit_points
            board.save()
        
        if commit.user:
            player, created = Player.objects.select_for_update().get_or_create(
                game=self, user=commit.user, 
                defaults={'points': self.project_points + self.commit_points})
            player.boards.add(board)
            if not created:
                # we need to get the total points for the user
                boards = player.boards.all().count() * self.project_points
                commits = Commit.objects.filter(
                    user=commit.user,
                    timestamp__gte=self.start,
                    timestamp__lte=self.end).count() * self.commit_points
                # TODO (rmyers): Add in problem points
                player.points = boards + commits
                player.save()
                

class Player(models.Model):
    """A player in the game."""
    
    game = models.ForeignKey(Game)
    user = models.ForeignKey(settings.AUTH_USER_MODEL)
    points = models.IntegerField(default=0)
    boards = models.ManyToManyField('Board')

    class Meta:
        ordering = ['-points']

    def __unicode__(self):
        return self.user

class Board(models.Model):
    """A project with commits in the game."""
    
    game = models.ForeignKey(Game)
    project = models.ForeignKey(Project)
    points = models.IntegerField(default=0)

    class Meta:
        ordering = ['-points']

    def __unicode__(self):
        return self.project

@receiver(post_save, sender=Commit)
def add_commit(sender, **kwargs):
    """Listens for new commits and adds them to the game."""
    active_game = Game.active()
    if active_game is not None:
        commit = kwargs.get('instance')
        from_orphan = not kwargs.get('created', False)
        active_game.add_commit(commit, from_orphan=from_orphan)