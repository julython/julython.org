import json
import logging

from django.core.management.base import BaseCommand, CommandError

from july.people.models import Commit, Project

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
                    Commit.create_by_email(commit['email'], commit, project)
                except Exception, e:
                    logging.exception("Error: %s" % e)