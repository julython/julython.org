import json
import logging

from django.shortcuts import render_to_response
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext

from july.api import to_dict
from july.live.models import Message, Connection
from django import http

@login_required
def live(request):
    if not request.session.get('live_token'):
        # TODO: maybe more random??
        request.session['live_token'] = request.user.key.urlsafe()
        
    token = request.session['live_token']
    models = Message.query().order(-Message.timestamp).fetch(10)
    m_list = [to_dict(m) for m in models]
    messages = json.dumps(m_list)
    
    return render_to_response('live/index.html', {
        'token': token, 'mesages': messages},
        context_instance=RequestContext(request))
    
def connected(request):
    logging.error('here')
    client_id = request.POST.get('from')

    connection = Connection.get_or_insert(client_id, client_id=client_id)
    connection.put()
    
    return http.HttpResponse('Ok')

def disconnected(request):
    
    client_id = request.POST.get('from')
    
    connection = Connection.get_by_id(client_id)
    if connection is None:
        return http.HttpResponse('Ok')
    
    connection.delete()
    
    return http.HttpResponse('Ok')