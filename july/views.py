
from django.views.decorators.cache import cache_page
from django.shortcuts import render_to_response
from django.template import RequestContext

from july.pages.models import Section

@cache_page(60)
def index(request):
    """Render the home page"""
    
    sections = Section.all().order('order')
    
    return render_to_response('index.html', RequestContext(request, {
        'sections': sections,
    }))