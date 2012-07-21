import json
import logging

from google.appengine.api import memcache
from google.appengine.api import channel

from django.shortcuts import render_to_response
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext
from django import http

from july.api import to_dict
from july.live.models import Message, Connection
from july.people.models import Accumulator

@login_required
def live(request):
    stats = Accumulator.get_histogram('global')
    total = sum(stats)
    message_future = Message.query().order(-Message.timestamp).fetch_async(30)
        
    # Julython live stuffs
    token_key = 'live_token:%s' % request.user.username
    token = memcache.get(token_key)
    if token is None:
        token = channel.create_channel(request.user.username)
        memcache.set(token_key, token, time=7000)

    message_models = message_future.get_result()
    
    m_list = [to_dict(m) for m in message_models]
    m_list.reverse()
    messages = json.dumps(m_list)
    
    return render_to_response('live/index.html', {
        'token': token, 'messages': messages, 'total': total},
        context_instance=RequestContext(request))

@login_required
def reconnect(request):
    """Endpoint for channel api to reconnect on token timeout."""
    
    token_key = 'live_token:%s' % request.user.username
    logging.info('reconnecting token: %s', token_key)
    token = channel.create_channel(request.user.username)
    memcache.set(token_key, token, time=7000)
    
    response_data = {'token': token}
    
    return http.HttpResponse(json.dumps(response_data), mimetype="application/json")
    
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