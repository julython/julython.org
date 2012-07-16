import webapp2
import logging
import datetime

from google.appengine.ext import ndb
from july.live.models import Connection

class Connect(webapp2.RequestHandler):
    
    def post(self):
        client_id = self.request.params.get('from')  
        logging.info("%s connected", client_id)

        connection = Connection.get_or_insert(client_id, client_id=client_id)
        connection.put()

class Disconnect(webapp2.RequestHandler):
    
    def post(self):
        client_id = self.request.params.get('from')
        logging.info("%s disconnected", client_id)

        key = ndb.Key('Connection', client_id)
        key.delete()

###
### Setup the routes for the Crontab
###
routes = [
    webapp2.Route('/_ah/channel/connected/', Connect),
    webapp2.Route('/_ah/channel/disconnected/', Disconnect),
] 

# The Main Application
app = webapp2.WSGIApplication(routes)