import datetime

from django.views.generic import list, detail

from july.game.models import Player, Game, Board, LanguageBoard
from july.people.models import Project, Location, Team, Language


class GameMixin(object):
    model = None

    def get_game(self):
        year = int(self.kwargs.get('year', 0))
        month = int(self.kwargs.get('month', 0))
        if not year:
            game = Game.active()
        else:
            date = datetime.datetime(year=year, month=month, day=15)
            game = Game.active(now=date)
        if game is None:
            game = Game.objects.latest()
        return game

    def get_queryset(self):
        game = self.get_game()
        return self.model.objects.filter(game=game).select_related()


class PlayerList(list.ListView, GameMixin):
    model = Player
    paginate_by = 100


class BoardList(list.ListView, GameMixin):
    model = Board
    paginate_by = 100


class LanguageBoardList(list.ListView, GameMixin):
    model = LanguageBoard
    paginate_by = 100


class ProjectView(detail.DetailView):
    model = Project


class LanguageView(detail.DetailView):
    model = Language

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
