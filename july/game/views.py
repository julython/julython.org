import datetime
import logging
from pytz import UTC

from django.views.generic import list, detail
from django.http.response import HttpResponse
from django.http import Http404
from django.views.decorators.csrf import csrf_exempt
from django.views.generic.list import ListView

from july.game.models import Player, Game, Board, LanguageBoard
from july.people.models import Project, Location, Team, Language


class GameMixin(object):

    def get_game(self):
        year = int(self.kwargs.get('year', 0))
        mon = int(self.kwargs.get('month', 0))
        day = self.kwargs.get('day')
        if day is None:
            day = 15
        day = int(day)
        if not all([year, mon]):
            now = None
        else:
            now = datetime.datetime(year=year, month=mon, day=day, tzinfo=UTC)
            logging.debug("Getting game for date: %s", now)
        return Game.active_or_latest(now=now)


class PlayerList(ListView, GameMixin):
    model = Player
    paginate_by = 100

    def get_queryset(self):
        game = self.get_game()
        return Player.objects.filter(
            game=game, user__is_active=True).select_related()


class BoardList(ListView, GameMixin):
    model = Board
    paginate_by = 100

    def get_queryset(self):
        game = self.get_game()
        return Board.objects.filter(
            game=game, project__active=True).select_related()


class LanguageBoardList(list.ListView, GameMixin):
    model = LanguageBoard
    paginate_by = 100


class ProjectView(detail.DetailView):
    model = Project


class LanguageView(detail.DetailView):
    model = Language


class LocationCollection(ListView, GameMixin):
    model = Location

    def get_queryset(self):
        game = self.get_game()
        return game.locations


class LocationView(detail.DetailView):
    model = Location

    def get_object(self):
        obj = super(LocationView, self).get_object()
        if not obj.approved:
            raise Http404("Location not found")
        return obj


class TeamCollection(ListView, GameMixin):
    model = Team

    def get_queryset(self):
        game = self.get_game()
        return game.teams


class TeamView(detail.DetailView):
    model = Team

    def get_object(self):
        obj = super(TeamView, self).get_object()
        if not obj.approved:
            raise Http404("Team not found")
        return obj


@csrf_exempt
def events(request, action, channel):
    logging.info('%s on %s', action, channel)
    if request.method == 'POST':
        logging.info(request.body)
    return HttpResponse('ok')
