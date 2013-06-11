import json

from django.contrib.auth.decorators import login_required
from django.contrib.auth import login as auth_login
from django.shortcuts import render_to_response, render, redirect
from django.template import Context
from django.conf import settings
from django.http import HttpResponseRedirect

from july.game.models import Game
from july.forms import RegistrationForm


def index(request):
    """Render the home page"""
    game = Game.active_or_latest()
    stats = game.histogram

    ctx = Context({
        'stats': json.dumps(stats),
        'game': game,
        'total': sum(stats),
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})

    return render_to_response('index.html', context_instance=ctx)


def help_view(request):
    """Render the help page"""
    ctx = Context({
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})

    return render_to_response('help.html', context_instance=ctx)


def register(request):
    """Register a new user"""
    if request.POST:
        form = RegistrationForm(request.POST)
        if form.is_valid():
            user = form.save(commit=False)
            user.save()
            # To login immediately after registering
            user.backend = "django.contrib.auth.backends.ModelBackend"
            auth_login(request, user)
            return redirect('july.views.index')
    else:
        form = RegistrationForm()

    return render(
        request,
        'registration/register.html', {'form': form})


@login_required
def login_redirect(request):
    if request.user.is_authenticated():
        return HttpResponseRedirect('/%s' % request.user.username )
    return HttpResponseRedirect('/')


def maintenance(request):
    """Site is down for maintenance, display this view for all."""
    ctx = Context({})

    return render_to_response('maintenance.html', context_instance=ctx)


def live(request):
    """Render the live view."""
    game = Game.active_or_latest()

    ctx = Context({
        'game': game,
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})

    return render_to_response('live/index.html', context_instance=ctx)
