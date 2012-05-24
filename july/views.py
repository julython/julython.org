
from django.shortcuts import render_to_response
from django.template import Context
from django.conf import settings

from july.pages.models import Section
from july.decorators import cache_page_anonymous

@cache_page_anonymous(60)
def index(request):
    """Render the home page"""
    
    sections = Section.all().order('order')
    
    ctx = Context({
        'sections': sections,
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})
    
    return render_to_response('index.html', context_instance=ctx)

def warmup(request):
    """Fire up the servers!"""
    from django.http import HttpResponse
    return HttpResponse('OK')