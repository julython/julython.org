import datetime
import logging

from django.http.response import HttpResponse
from django.views.decorators.csrf import csrf_exempt
from django.views.generic import detail
from django.views.generic.list import ListView

from july.game.models import Player, Game, Board
from july.people.models import Project, Location, Team


class GameMixin(object):

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


class PlayerList(ListView, GameMixin):
    model = Player
    paginate_by = 100

    def get_queryset(self):
        game = self.get_game()
        return Player.objects.filter(game=game).select_related()


class BoardList(ListView, GameMixin):
    model = Board
    paginate_by = 100

    def get_queryset(self):
        game = self.get_game()
        return Board.objects.filter(game=game).select_related()


class ProjectView(detail.DetailView):
    model = Project


class LocationCollection(ListView, GameMixin):
    model = Location

    def get_queryset(self):
        game = self.get_game()
        return game.locations


class LocationView(detail.DetailView):
    model = Location


class TeamCollection(ListView, GameMixin):
    model = Team

    def get_queryset(self):
        game = self.get_game()
        return game.teams


class TeamView(detail.DetailView):
    model = Team


@csrf_exempt
def events(request, action, channel):
    logging.info('%s on %s', action, channel)
    if request.method == 'POST':
        logging.info(request.body)
    return HttpResponse('ok')
