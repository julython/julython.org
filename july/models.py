"""
Custom User Model for Julython
==============================

This is experimental, but so much worth it!
"""

from django.db import models
from django.contrib.auth.models import AbstractUser

from social_auth.models import UserSocialAuth

from july.people.models import Location, Team, Project


class User(AbstractUser):
    location = models.ForeignKey(Location, blank=True, null=True,
                                 related_name='location_members')
    team = models.ForeignKey(Team, blank=True, null=True,
                             related_name='team_members')
    projects = models.ManyToManyField(Project, blank=True, null=True)
    description = models.TextField(blank=True)
    url = models.URLField(blank=True, null=True)
    picture_url = models.URLField(blank=True, null=True)

    def __unicode__(self):
        return self.get_full_name() or self.username

    def add_auth_id(self, auth_str):
        """
        Add a social auth identifier for this user.

        The `auth_str` should be in the format '{provider}:{uid}'
        this is useful for adding multiple unique email addresses.

        Example::

            user = User.objects.get(username='foo')
            user.add_auth_id('email:foo@example.com')
        """
        provider, uid = auth_str.split(':')
        UserSocialAuth.create_social_auth(self, uid, provider)

    def get_provider(self, provider):
        """Return the uid of the provider or None if not set."""
        try:
            return self.social_auth.filter(provider=provider).get()
        except UserSocialAuth.DoesNotExist:
            return None

    @property
    def gittip(self):
        return self.get_provider('gittip')

    @property
    def twitter(self):
        return self.get_provider('twitter')

    @property
    def github(self):
        return self.get_provider('github')

    @classmethod
    def get_by_auth_id(cls, auth_str):
        """
        Return the user identified by the auth id.

        Example::

            user = User.get_by_auth_id('twitter:julython')
        """
        provider, uid = auth_str.split(':')
        sa = UserSocialAuth.get_social_auth(provider, uid)
        if sa is None:
            return None
        return sa.user

    @property
    def auth_ids(self):
        auths = self.social_auth.all()
        return [':'.join([a.provider, a.uid]) for a in auths]

    @property
    def points(self):
        try:
            player = self.player_set.latest()
        except:
            return 0
        return player.points

    @property
    def total(self):
        return self.points
