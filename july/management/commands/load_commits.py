import json
import logging
from datetime import datetime

import pytz

from django.core.management.base import BaseCommand
from django.core.management.base import CommandError
from django.utils.timezone import make_aware

from july.people.models import Commit, Project

def to_datetime(ts):
    d = datetime.fromtimestamp(ts)
    t = make_aware(d, pytz.UTC)
    return t

def to_commit(commit):
    new = {}
    attrs = ['hash', 'author', 'name', 'message', 'url', 'email']
    new['timestamp'] = to_datetime(commit['timestamp'])
    new['created_on'] = to_datetime(commit['created_on'])
    for key in attrs:
        new[key] = commit.get(key) or ''
    
    return new

class Command(BaseCommand):
    args = '<commits.json>'
    help = 'Load commits from json file'

    def handle(self, *args, **options):
        if len(args) != 1:
            raise CommandError('Must supply a JSON file of commits.')
        
        with open(args[0], 'r') as commit_file:
            commits = json.loads(commit_file.read())
            for commit in commits['models']:
                try:
                    project = Project.objects.get(url=commit['project'])
                    c = to_commit(commit)
                    Commit.create_by_email(c['email'], c, project)
                except Exception:
                    logging.exception("Error: %s" % commit)