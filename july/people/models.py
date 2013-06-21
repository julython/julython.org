import logging
import json
from urlparse import urlparse

from django.db import models, transaction
from django.conf import settings
from django.template.defaultfilters import slugify
from django.core.urlresolvers import reverse
from django.db.models.aggregates import Sum


class Commit(models.Model):
    """
    Commit record for the profile, the parent is the profile
    that way we can update the commit count and last commit timestamp
    in the same transaction.
    """
    user = models.ForeignKey(settings.AUTH_USER_MODEL, blank=True, null=True)
    hash = models.CharField(max_length=255, unique=True)
    author = models.CharField(max_length=255, blank=True)
    name = models.CharField(max_length=255, blank=True)
    email = models.CharField(max_length=255, blank=True)
    message = models.CharField(max_length=2024, blank=True)
    url = models.CharField(max_length=512, blank=True)
    project = models.ForeignKey("Project", blank=True, null=True)
    timestamp = models.DateTimeField()
    created_on = models.DateTimeField(auto_now_add=True)
    files = models.TextField(blank=True, null=True)

    class Meta:
        ordering = ['-timestamp']

    def __str__(self):
        return self.__unicode__()

    def __unicode__(self):
        return u'Commit: %s' % self.hash

    @classmethod
    def create_by_email(cls, email, commits, project=None):
        """Create a commit by email address"""
        return cls.create_by_auth_id('email:%s' % email, commits, project=project)

    @classmethod
    def user_model(cls):
        return cls._meta.get_field('user').rel.to

    @classmethod
    def create_by_auth_id(cls, auth_id, commits, project=None):
        if not isinstance(commits, (list, tuple)):
            commits = [commits]

        user = cls.user_model().get_by_auth_id(auth_id)

        if user:
            return cls.create_by_user(user, commits, project=project)
        return cls.create_orphan(commits, project=project)

    @classmethod
    @transaction.commit_on_success
    def create_by_user(cls, user, commits, project=None):
        """Create a commit with parent user, updating users points."""
        created_commits = []

        for c in commits:
            c['user'] = user
            c['project'] = project
            commit_hash = c.pop('hash', None)
            files = c.pop('files', [])
            try:
                c['files'] = json.dumps(files)
            except:
                pass

            languages = c.pop('languages', [])
            # TODO: (rmyers) do something with files, languages
            if commit_hash is None:
                logging.info("Commit hash missing in create.")
                continue
            commit, created = cls.objects.get_or_create(hash=commit_hash,
                defaults=c
            )
            if created:
                # increment the counts
                created_commits.append(commit)
            else:
                commit.user = user
                commit.save()

        # Check if there are no new commits and return
        if not created_commits:
            return []

        if project is not None:
            user.projects.add(project)
            user.save()

        # TODO: (Rober Myers) add a call to the defer a task to calculate
        # game stats in a queue?
        return created_commits

    @classmethod
    def create_orphan(cls, commits, project=None):
        """Create a commit with no parent."""
        created_commits = []
        for c in commits:
            c['project'] = project
            commit_hash = c.get('hash')
            if commit_hash is None:
                logging.info("Commit hash missing in create.")
                continue
            files = c.pop('files', [])
            try:
                c['files'] = json.dumps(files)
            except:
                pass

            languages = c.pop('languages', [])
            # TODO: (rmyers) do something with files, languages

            commit, created = cls.objects.get_or_create(
                hash=commit_hash,
                defaults=c
            )
            if created:
                created_commits.append(commit)

        return created_commits


class Project(models.Model):
    """
    Project Model:

    This is either a brand new project or an already existing project
    such as #django, #fabric, #tornado, #pip, etc.

    When a user Tweets a url we can automatically create anew project
    for any of the repo host we know already. (github, bitbucket)
    """

    url = models.CharField(max_length=255)
    description = models.TextField(blank=True)
    name = models.CharField(max_length=255, blank=True)
    forked = models.BooleanField(default=False)
    forks = models.IntegerField(default=0)
    watchers = models.IntegerField(default=0)
    parent_url = models.CharField(max_length=255, blank=True)
    created_on = models.DateTimeField(auto_now_add=True)
    slug = models.SlugField()
    service = models.CharField(max_length=30, blank=True, default='')
    repo_id = models.IntegerField(blank=True, null=True)

    def __unicode__(self):
        if self.name:
            return self.name
        else:
            return self.slug

    def save(self, *args, **kwargs):
        self.slug = self.project_name
        super(Project, self).save(*args, **kwargs)

    @property
    def points(self):
        try:
            board = self.board_set.latest()
        except:
            return 0
        return board.points

    @property
    def total(self):
        return self.points

    @property
    def project_name(self):
        return self.parse_project_name(self.url)

    def get_absolute_url(self):
        return reverse('project-details', args=[self.slug])

    @classmethod
    def create(cls, **kwargs):
        """Get or create shortcut."""
        repo_id = kwargs.get('repo_id')
        url = kwargs.get('url')
        slug = cls.parse_project_name(url)
        service = kwargs.get('service')

        # If the repo is on a service with no repo id, we can't handle renames.
        if not repo_id:
            project, created = cls.objects.get_or_create(
                slug=slug, defaults=kwargs)

        # Catch renaming of the repo.
        else:
            query = cls.objects.filter(service=service, repo_id=repo_id)
            if query.count() == 1:
                project = query[0]
                created = False
            else:
                project, created = cls.objects.get_or_create(
                    slug=slug, defaults=kwargs)

        # Update stale project information.
        if not created:
            cls.objects.filter(pk=project.pk).update(slug=slug, **kwargs)

        return project

    @staticmethod
    def parse_project_name(url):
        """
        Parse a project url and return a name for it.

        Example::

            Given:
              http://github.com/julython/julython.org
            Return:
              gh-julython-julython.org

        This is used as the Key name in order to speed lookups during
        api requests.
        """
        if not url:
            return
        hosts_lookup = {
            'github.com': 'gh',
            'bitbucket.org': 'bb',
        }
        parsed = urlparse(url)
        path = parsed.path
        if path.startswith('/'):
            path = path[1:]
        tokens = path.split('/')
        netloc_slug = parsed.netloc.replace('.', '-')
        host_abbr = hosts_lookup.get(parsed.netloc, netloc_slug)
        name = '-'.join(tokens)
        if name.endswith('-'):
            name = name[:-1]
        name = name.replace('.', '_')
        return '%s-%s' % (host_abbr, name)


class AchievedBadge(models.Model):
    user = models.ForeignKey(settings.AUTH_USER_MODEL, blank=True, null=True)
    badge = models.ForeignKey("Badge", blank=True, null=True)
    achieved_on = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return self.__unicode__()

    def __unicode__(self):
        return u'%s: %s' % (self.user, self.badge)


class Badge(models.Model):
    name = models.CharField(max_length=255, blank=True)
    text = models.CharField(max_length=255, blank=True)
    description = models.CharField(max_length=2024, blank=True)

    def __str__(self):
        return self.name

    def __unicode__(self):
        return self.name


class Group(models.Model):
    total = models.IntegerField(default=0)
    name = models.CharField(max_length=64, blank=False)
    slug = models.SlugField(primary_key=True)

    class Meta:
        abstract=True

    def __str__(self):
        return self.name

    def __unicode__(self):
        return self.name

    def members_by_points(self):
        raise NotImplementedError("members_by_points must be implemented by the subclass!")

    def total_points(self):
        raise NotImplementedError("total_points must be implemented by the subclass!")

    @classmethod
    def create(cls, name):
        slug = slugify(name)
        return cls.objects.get_or_create(slug=slug, defaults={'name': name})


class Location(Group):
    """Simple model for holding point totals and projects for a location"""

    def members_by_points(self):
        from july.game.models import Game
        latest = Game.active_or_latest()
        return latest.players.filter(location=self).order_by('-player__points')

    def total_points(self):
        from july.game.models import Game, Player
        latest = Game.active_or_latest()
        query = Player.objects.filter(user__location=self, game=latest)
        total = query.aggregate(Sum('points'))
        points = total.get('points__sum')
        return points or 0

    def get_absolute_url(self):
        from django.core.urlresolvers import reverse
        return reverse('member-list', kwargs={'location_slug': self.slug})


class Team(Group):
    """Simple model for holding point totals and projects for a Team"""

    def members_by_points(self):
        from july.game.models import Game
        latest = Game.active_or_latest()
        return latest.players.filter(team=self).order_by('-player__points')

    def total_points(self):
        from july.game.models import Game, Player
        latest = Game.active_or_latest()
        query = Player.objects.filter(user__team=self, game=latest)
        total = query.aggregate(Sum('points'))
        points = total.get('points__sum')
        return points or 0


    def get_absolute_url(self):
        from django.core.urlresolvers import reverse
        return reverse('team-details', kwargs={'team_slug': self.slug})
