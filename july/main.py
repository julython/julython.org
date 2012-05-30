import logging

from flask import Flask, render_template, session, g, redirect, url_for, request, abort
from werkzeug.debug import DebuggedApplication

from july import settings
from gae_django.auth import oauth
from gae_django.auth.models import User


app = Flask('julython')
app.config.from_object(settings)

# Wrap the applictaion in middleware for debugging fun
app.wsgi_app = DebuggedApplication(app.wsgi_app, evalex=settings.DEBUG)

def login_required(f):
    """Checks whether user is logged in or raises error 401."""
    def decorator(*args, **kwargs):
        if not g.user:
            return redirect(url_for('twitter_signin'))
        return f(*args, **kwargs)
    return decorator

@app.before_request
def before_request():
    """Set the logged in user, this will be None if not found."""
    g.user = None
    if 'username' in session:
        g.user = User.all().filter('username', session.get('username')).get()

@app.route('/')
def index():
    from july.pages.models import Section
    logging.error(request.headers)
    sections = Section.all().fetch(100)
    return render_template('index.html', sections=sections, user=g.user)

@app.route('/me/')
@login_required
def profile():
    from july.records import Record
    cursor = request.args.get('cursor', None)
    
    records = Record.all().ancestor(g.user)
    
    if cursor:
        records.with_cursor(cursor)
    
    recs = records.fetch(35)
    return render_template('me.html', records=recs, user=g.user)

@app.route('/signin/')
def twitter_signin():
    consumer_key = app.config['TWITTER_CONSUMER_KEY']
    consumer_secret = app.config['TWITTER_CONSUMER_SECRET']
    callback_url = app.config['TWITTER_CALLBACK']
        
    client = oauth.TwitterClient(consumer_key, consumer_secret, callback_url)
    return redirect(client.get_authenticate_url())

@app.route('/accounts/twitter/verify/')
def twitter_verify():
    auth_token = request.args.get("oauth_token")
    auth_verifier = request.args.get("oauth_verifier")
    
    consumer_key = settings.TWITTER_CONSUMER_KEY
    consumer_secret = settings.TWITTER_CONSUMER_SECRET
    callback_url = settings.TWITTER_CALLBACK
    
    client = oauth.TwitterClient(consumer_key, consumer_secret, callback_url)
    user_info = client.get_user_info(auth_token, auth_verifier=auth_verifier)

    g.user = User.from_twitter_info(user_info)
    session['username'] = g.user.username
    
    return redirect(settings.LOGIN_REDIRECT_URL)

@app.route('/signout/')
def signout():
    session['username'] = None
    return redirect(url_for('index'))

# TODO: this should be removed 
# For debugging the app
@app.route('/exception/')
def exception():
    raise Exception