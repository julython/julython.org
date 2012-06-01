#import logging
from django.contrib.auth.decorators import login_required
from django.shortcuts import render_to_response
from django.template import Context
from django.conf import settings
from django.core.cache import cache
from django.http import HttpResponseRedirect

from july.pages.models import Section

def index(request):
    """Render the home page"""
    
    sections = cache.get('front_page')
    if sections is None:
        sections = Section.all().order('order').fetch(10)
        cache.set('front_page', sections, 120)
        
    ctx = Context({
        'sections': sections,
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})
    
    return render_to_response('index.html', context_instance=ctx)

@login_required
def login_redirect(request):
    if request.user != None:
        return HttpResponseRedirect('/%s'%request.user.username )
    return HttpResponseRedirect('/')
    
     

def warmup(request):
    """Fire up the servers!"""
    from django.http import HttpResponse
    return HttpResponse('OK')
