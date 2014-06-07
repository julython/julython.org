import datetime
import logging
from pytz import UTC

from django.views.generic import list, detail
from django.http.response import HttpResponse
from django.http import Http404
from django.views.decorators.csrf import csrf_exempt
from django.views.generic.list import ListView
from django.views.generic import View
from django.shortcuts import render

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


class GameBoard(View, GameMixin):
    template_name = 'game/game_list.html'

    def get(self, request, *args, **kwargs):
        game = self.get_game()
        people = Player.objects.filter(
            game=game, user__is_active=True).select_related()
        ctx = {
            'people': people,
            'teams': game.teams,
            'locations': game.locations
        }
        return render(request, self.template_name, ctx)


class PlayerList(ListView, GameMixin):
    model = Player
    paginate_by = 100

    def get_queryset(self):
        game = self.get_game()
        return Player.objects.filter(
            game=game, user__is_active=True).select_related()


class BoardList(View, GameMixin):
    template_name = 'game/board_list.html'

    def get(self, request, *args, **kwargs):
        game = self.get_game()
        small_boards = Board.objects.filter(
            game=game, project__watchers__lt=10,
            project__active=True).select_related('project')
        medium_boards = Board.objects.filter(
            game=game, project__watchers__gte=10,
            project__watchers__lt=100,
            project__active=True).select_related('project')
        large_boards = Board.objects.filter(
            game=game, project__watchers__gte=100,
            project__active=True).select_related('project')
        ctx = {
            'small_boards': small_boards,
            'medium_boards': medium_boards,
            'large_boards': large_boards,
        }
        return render(request, self.template_name, ctx)


class LanguageBoardList(list.ListView, GameMixin):
    model = LanguageBoard
    paginate_by = 100


class ProjectView(detail.DetailView):
    model = Project

    def get_queryset(self):
        return self.model.objects.filter(active=True)


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
