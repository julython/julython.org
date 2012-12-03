
from django.db import models
from django.conf import settings

class Game(models.Model):
    
    start = models.DateTimeField()
    end = models.DateTimeField()
    players = models.ManyToManyField(settings.AUTH_USER_MODEL, through='Player')
    
    def __unicode__(self):
        if self.end.month == 8:
            return 'Julython %s' % self.end.year
        else:
            return 'J(an)ulython %s' % self.end.year

class Player(models.Model):
    """A player in the game."""
    
    game = models.ForeignKey(Game)
    user = models.ForeignKey(settings.AUTH_USER_MODEL)
    points = models.IntegerField(default=0)