import json

from google.appengine.ext import ndb

from django.contrib.auth.decorators import login_required
from django.shortcuts import render_to_response
from django.template import Context
from django.conf import settings
from django.http import HttpResponseRedirect

from gae_django.auth.models import User

from july.people.models import Accumulator, Location, Project, Team
from google.appengine.api import channel
from july.live.models import Message
from july.api import to_dict


def index(request):
    """Render the home page"""
    
    # For now we are just using hard coded sections
    #sections = cache.get('front_page')
    #if sections is None:
    #    sections = Section.all().order('order').fetch(10)
    #    cache.set('front_page', sections, 120)
    
    stats = []
    total = 0
    people = []
    locations = []
    projects = []
    teams = []
    messages = []
    token = ''
    
    # this is only shown on authenticated page loads
    # to save on the overhead. 
    if request.user.is_authenticated():
        stats = Accumulator.get_histogram('global')
        total = sum(stats)
        location_future = Location.query().order(-Location.total).fetch_async(3)
        people_future = User.query().order(-ndb.GenericProperty('total')).fetch_async(5)
        project_future = Project.query().order(-Project.total).fetch_async(5)
        team_future = Team.query().order(-Team.total).fetch_async(3)
        locations = location_future.get_result()
        people = people_future.get_result()
        projects = project_future.get_result()
        teams = team_future.get_result()
        
        # Julython live stuffs
        if not request.session.get('live_token'):
            _token = channel.create_channel(request.user.username)
            request.session['live_token'] = _token
            
        token = request.session['live_token']
        message_future = Message.query().order(-Message.timestamp).fetch_async(10)
        message_models = message_future.get_result()
        m_list = [to_dict(m) for m in message_models]
        messages = json.dumps(m_list)
    
    ctx = Context({
        'sections': [],
        'people': people,
        'projects': projects,
        'locations': locations,
        'teams': teams,
        'stats': json.dumps(stats),
        'total': total,
        'token': token,
        'messages': messages,
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
