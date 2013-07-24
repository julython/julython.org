
from django.core.management.base import BaseCommand, CommandError

from july.models import User
from july.auth.social import merge_users
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

        merge_users(new_user, old_user, commit)
