
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

#
# Location totals SQL
# 
# select july_user.location_id, 
#        sum(game_player.points) as points 
#        from game_player 
#        left join july_user 
#        where game_player.user_id = july_user.id 
#        group by july_user.location_id 
#        order by points DESC;
#
# Team totals SQL
# select july_user.team_id, 
#        sum(game_player.points) as points 
#        from game_player 
#        left join july_user 
#        where game_player.user_id = july_user.id 
#        group by july_user.team_id 
#        order by points DESC;
