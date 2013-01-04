import json

from django.contrib.auth.decorators import login_required
from django.shortcuts import render_to_response
from django.template import Context
from django.conf import settings
from django.http import HttpResponseRedirect

from july.game.models import Game


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

@login_required
def login_redirect(request):
    if request.user.is_authenticated():
        return HttpResponseRedirect('/%s'%request.user.username )
    return HttpResponseRedirect('/')

def maintenance(request):
    """Site is down for maintenance, display this view for all."""
    ctx = Context({})
    
    return render_to_response('maintenance.html', context_instance=ctx)

def warmup(request):
    """Fire up the servers!"""
    from django.http import HttpResponse
    return HttpResponse('OK')
