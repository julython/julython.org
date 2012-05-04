
from django.shortcuts import render_to_response
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext

@login_required
def member_profile(request):
    """Member profile page."""
    return render_to_response('people/profile.html', RequestContext(request, {}))