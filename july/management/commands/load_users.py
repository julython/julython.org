import json
import logging
import requests
from time import sleep

from django.core.management.base import BaseCommand, CommandError
from django.template.defaultfilters import slugify

from july.models import User
from july.people.models import Location, Team
from optparse import make_option

def get_twitter_id(username):
    # don't overload twitter api
    sleep(60)
    resp = requests.get('https://api.twitter.com/1/users/lookup.json?screen_name=%s' % username)
    data = json.loads(resp.content)
    return data[0]

def get_location(location):
    if location is None:
        return
    slug = slugify(location)
    loc, _ = Location.objects.get_or_create(slug=slug, defaults={'name': location})
    return loc

def get_team(team):
    if team is None:
        return
    slug = slugify(team)
    t, _ = Team.objects.get_or_create(slug=slug, defaults={'name': team})
    return t

class FakeUser(object):
    
    def __init__(self, user, commit=False):
        self.username = None
        self.first_name = user.get('first_name', '')
        self.last_name = user.get('last_name', '')
        self.password = '!'
        self._auth_ids = user.get('auth_ids', [])
        self.url = user.get('url', '') or ''
        self.location = get_location(user.get('location'))
        self.team = get_team(user.get('team'))
        self.description = user.get('description', '') or ''
        self.picture_url = user.get('picture_url', '')
        self.auth_ids = []
        for auth in self._auth_ids:
            provider, uid = auth.split(':')
            if provider == 'own':
                self.username = uid
            elif provider == 'twitter':
                if commit:
                    data = get_twitter_id(uid)
                    tid = data['id']
                    self.picture_url = data.get('profile_image_url', '')
                else:
                    tid = uid
                self.auth_ids.append('twitter:%s' % tid)
            else:
                self.auth_ids.append(auth)
    
    def create(self):
        if self.username is None:
            print self.__dict__
            return
        defaults={
            'first_name': self.first_name,
            'last_name': self.last_name,
            'url': self.url,
            'picture_url': self.picture_url,
            'description': self.description,
            'team': self.team,
            'location': self.location
        }
        user, created = User.objects.get_or_create(
            username=self.username, defaults=defaults)
        if not created:
            for k, v in defaults.iteritems():
                setattr(user, k, v)
            user.save()
        user_auth_ids = user.auth_ids
        for auth in self.auth_ids:
            if auth not in user_auth_ids:
                user.add_auth_id(auth)

class Command(BaseCommand):
    args = '<user.json>'
    help = 'Load users from json file'
    option_list = BaseCommand.option_list + (
        make_option('--commit',
            action='store_true',
            dest='commit',
            default=False,
            help='Actually poll twitter/github for account info.'),
    )
    
    def handle(self, *args, **options):
        if len(args) != 1:
            raise CommandError('Must supply a JSON file of users.')
        
        with open(args[0], 'r') as user_file:
            users = json.loads(user_file.read())
            total = len(users['models'])
            count = 0
            for user in users['models']:
                count += 1
                try:
                    f = FakeUser(user, options['commit'])
                    f.create()
                    logging.info("Loaded %s of %s: %s", count, total, f.username)
                except Exception, e:
                    logging.exception("Error: %s" % e)