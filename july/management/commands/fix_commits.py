import logging

from django.core.management.base import BaseCommand

from july.people.models import Commit
from july.models import User
from optparse import make_option


class Command(BaseCommand):
    args = ''
    help = 'Associate orphan commits'
    option_list = BaseCommand.option_list + (
        make_option(
            '--commit',
            action='store_true',
            dest='commit',
            default=False,
            help='Actually change the commits.'),
    )

    def handle(self, *args, **options):
        commit = options['commit']

        bad_emails = []
        fixed = 0
        if not commit:
            count = Commit.objects.filter(user=None)
            logging.info("Found %s commits, add --commit to fix", count)
            return

        for commit in Commit.objects.filter(user=None):
            if commit.email in bad_emails or not commit.email:
                continue
            user = User.get_by_auth_id('email:%s' % commit.email)
            if not user:
                logging.info("Found bad email: %s", commit.email)
                bad_emails.append(commit.email)
                continue
            commit.user = user
            commit.save()
            user.projects.add(commit.project)
            fixed += 1
            if (fixed % 100) == 0:
                logging.info("Fixed %s commits", fixed)

        logging.info("Fixed %s commits", fixed)
