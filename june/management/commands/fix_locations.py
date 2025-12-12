
import logging

from django.core.management.base import BaseCommand
from django.template.defaultfilters import slugify

from july.models import User
from july.people.models import Location
from july.utils import check_location
from optparse import make_option


class Command(BaseCommand):
    help = 'fix locations'
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
        empty = 0
        fine = 0
        fixable = 0
        bad = []
        for location in Location.objects.all():
            user_count = User.objects.filter(location=location).count()
            if not user_count:
                logging.info("Empty location: %s", location)
                if commit:
                    location.delete()
                    logging.info('Deleted')
                empty += 1
                continue
            l = check_location(location.name)
            if l == location.name:
                logging.info('Location fine: %s', location)
                fine += 1
                continue

            if not commit:
                if l:
                    fixable += 1
                else:
                    bad.append((location, user_count))
                continue
            elif l is not None:
                new_loc = Location.create(l)
                User.objects.filter(location=location).update(location=new_loc)
                user_count = User.objects.filter(location=location).count()
                if not user_count:
                    logging.error("missed users!")
                else:
                    location.delete()
            elif l is None:
                logging.info('Bad location: %s', location)
                location.approved = False
                location.save()

        if not commit:
            [logging.error('Bad Loc: %s, count: %s', l, c) for l, c in bad]
            logging.info('Empty: %s, Fine: %s, fixable: %s',
                         empty, fine, fixable)
            logging.info('Add --commit to fix locations')
