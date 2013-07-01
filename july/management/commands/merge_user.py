import json
import logging
import requests
from time import sleep

from django.core.management.base import BaseCommand, CommandError
from django.template.defaultfilters import slugify

from july.models import User
from july.people.models import Location, Team, Commit
from july.game.models import Board, Player
from social_auth.models import UserSocialAuth
from optparse import make_option


class Command(BaseCommand):
    args = '<old_user.id> <new_user.id>'
    help = 'move all records from old user to new user'
    option_list = BaseCommand.option_list + (
        make_option(
            '--commit',
            action='store_true',
            dest='commit',
            default=False,
            help='Actually move the items.'),
    )

    def handle(self, *args, **options):
        commit = options['commit']
        if len(args) != 2:
            raise CommandError('You must enter two user ids.')
        try:
            old_user = User.objects.get(pk=int(args[0]))
            new_user = User.objects.get(pk=int(args[1]))
        except:
            raise CommandError("unable to find those users.")

        logging.info("Merging %s (%s) into: %s (%s)", old_user, old_user.id,
                     new_user, new_user.id)

        if not commit:
            for o in UserSocialAuth.objects.filter(user=old_user):
                logging.info("Old Auth: %s", o)
            logging.info("Found %s commits",
                         Commit.objects.filter(user=old_user).count())
            for p in Player.objects.filter(user=old_user):
                logging.info("Player: %s, %s points: %s", p, p.game, p.points)
            logging.info("Merge player by adding --commit")
        else:
            UserSocialAuth.objects.filter(user=old_user).update(user=new_user)
            for commit in Commit.objects.filter(user=old_user):
                commit.user = new_user
                commit.save()
            logging.info("Merged")
