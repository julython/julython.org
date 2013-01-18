import json
import logging
from django.http.response import HttpResponse
from django.views.decorators.csrf import csrf_exempt

@csrf_exempt
def events(request, action, channel):
    logging.info('%s on %s', action, channel)
    if request.method == 'POST':
        logging.info(request.body)
    return HttpResponse('ok')