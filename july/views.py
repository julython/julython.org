
from django.views.decorators.cache import cache_page
from django.shortcuts import render_to_response
from django.template import Context

from july.pages.models import Section

@cache_page(60)
def index(request):
    """Render the home page"""
    
    sections = Section.all().order('order')
    
    ctx = Context({
        'sections': sections,
        'MEDIA_URL': '/static/',
        'STATIC_URL': '/static/'})
    
    return render_to_response('index.html', context_instance=ctx)

def warmup(request):
    """Fire up the servers!"""
    from django.http import HttpResponse
    return HttpResponse('OK')