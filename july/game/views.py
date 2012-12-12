import datetime

from django.http import Http404
from django.views.generic import list, detail

from july.game.models import Player, Game, Board
from july.people.models import Project, Location, Team


class GameMixin(object):
    
    def get_game(self):
        year = int(self.kwargs.get('year', 0))
        month = int(self.kwargs.get('month', 0))
        if not year:
            game = Game.objects.latest()
        else:
            date = datetime.datetime(year=year, month=month, day=15)
            game = Game.active(now=date)
        if game is None:
            raise Http404
        return game
        

class PlayerList(list.ListView, GameMixin):
    model = Player
    paginate_by = 100
    
    def get_queryset(self):
        game = self.get_game()
        return Player.objects.filter(game=game)


class BoardList(list.ListView, GameMixin):
    model = Board
    paginate_by = 100
    
    def get_queryset(self):
        game = self.get_game()
        return Board.objects.filter(game=game)


class ProjectView(detail.DetailView):
    model = Project
    

class LocationCollection(list.ListView, GameMixin):
    model = Location
    
    def get_queryset(self):
        game = self.get_game()
        return game.locations

class LocationView(detail.DetailView):
    model = Location


class TeamCollection(list.ListView, GameMixin):
    model = Team
    
    def get_queryset(self):
        game = self.get_game()
        return game.teams

class TeamView(detail.DetailView):
    model = Team