import json
import logging

from django.core.management.base import BaseCommand, CommandError

from july.people.models import Commit, Project

class Command(BaseCommand):
    args = '<project.json>'
    help = 'Load projects from json file'

    def handle(self, *args, **options):
        if len(args) != 1:
            raise CommandError('Must supply a JSON file of projects.')
        
        with open(args[0], 'r') as project_file:
            projects = json.loads(project_file.read())
            for project in projects['models']:
                try:
                    Project.objects.get_or_create(
                        url=project['url'], 
                        description=project['description'],
                        name=project['name'])
                except Exception, e:
                    logging.exception("Error: %s" % e)