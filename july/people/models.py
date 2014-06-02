import logging
from urlparse import urlparse
from datetime import datetime, timedelta

from django.db import models, transaction
from django.conf import settings
from django.template.defaultfilters import slugify
from django.core.urlresolvers import reverse
from jsonfield import JSONField
from django.db.models.aggregates import Sum, Count
from django.utils.timezone import utc, now
from django.utils.html import strip_tags
from django.core.mail import mail_admins
from django.template import loader
import requests


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
    files = JSONField(blank=True, null=True)

    class Meta:
        ordering = ['-timestamp']

    def __str__(self):
        return self.__unicode__()

    def __unicode__(self):
        return u'Commit: %s' % self.hash

    @property
    def languages(self):
        langs = []
        if self.files:
            for f in self.files:
                langs.append(f.get('language'))
        langs = filter(None, langs)
        return set(langs)

    @classmethod
    def create_by_email(cls, email, commits, project=None):
        """Create a commit by email address"""
        return cls.create_by_auth_id(
            'email:%s' % email, commits, project=project)

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

        if not user.is_active:
            return created_commits

        for c in commits:
            c['user'] = user
            c['project'] = project
            commit_hash = c.pop('hash', None)

            if commit_hash is None:
                logging.info("Commit hash missing in create.")
                continue
            commit, created = cls.objects.get_or_create(
                hash=commit_hash,
                defaults=c)
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

        # TODO: (Robert Myers) add a call to the defer a task to calculate
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

            commit, created = cls.objects.get_or_create(
                hash=commit_hash,
                defaults=c)
            if created:
                created_commits.append(commit)

        return created_commits

    @classmethod
    def calendar(cls, game, **kwargs):
        """
        Returns number of commits per day for a date range.
        """
        count = cls.objects.filter(
            timestamp__range=(game.start, game.end), **kwargs) \
            .extra(select={'timestamp': 'date(timestamp)'}) \
            .values('timestamp').annotate(commit_count=Count('id'))
        resp = {
            'start': game.start.date(),
            'end': game.end.date(),
            'objects': list(count)
        }
        return resp


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
    updated_on = models.DateTimeField(auto_now=True)
    slug = models.SlugField()
    service = models.CharField(max_length=30, blank=True, default='')
    repo_id = models.IntegerField(blank=True, null=True)
    active = models.BooleanField(default=True)

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
            defaults = kwargs.copy()
            defaults['slug'] = slug
            project, created = cls.objects.get_or_create(
                service=service, repo_id=repo_id,
                defaults=defaults)
            if created and cls.objects.filter(slug=slug).count() > 1:
                # This is an old project that was created without a repo_id.
                project.delete()  # Delete the duplicate project
                project = cls.objects.get(slug=slug)

        if not project.active:
            # Don't bother updating this project and don't add commits.
            return None

        # Update stale project information.
        project.update(slug, created, **kwargs)
        return project

    @classmethod
    def _get_bitbucket_data(cls, **kwargs):
        """Update info from bitbucket if needed."""
        url = kwargs.get('url', '')
        parsed = urlparse(url)
        if parsed.netloc == 'bitbucket.org':
            # grab data from the bitbucket api
            # TODO: (rmyers) authenticate with oauth?
            api = 'https://bitbucket.org/api/1.0/repositories%s'
            try:
                r = requests.get(api % parsed.path)
                data = r.json()
                kwargs['description'] = data.get('description') or ''
                kwargs['forks'] = data.get('forks_count') or 0
                kwargs['watchers'] = data.get('followers_count') or 0
            except:
                logging.exception("Unable to parse: %s", url)
        return kwargs.iteritems()

    def update(self, slug, created, **kwargs):
        old = (now() - self.updated_on).seconds >= 21600
        if created or old or slug != self.slug:
            for key, value in self._get_bitbucket_data(**kwargs):
                setattr(self, key, value)
            self.slug = slug
            self.save()

    @classmethod
    def parse_project_name(cls, url):
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


class Group(models.Model):
    slug = models.SlugField(primary_key=True)
    name = models.CharField(max_length=64, blank=False)
    total = models.IntegerField(default=0)
    approved = models.BooleanField(default=False)
    rel_lookup = None
    lookup = None

    class Meta:
        abstract = True

    def __str__(self):
        return self.name

    def __unicode__(self):
        return self.name

    def members_by_points(self):
        from july.game.models import Game
        latest = Game.active_or_latest()
        kwargs = {
            self.lookup: self
        }
        return latest.players.filter(**kwargs).order_by('-player__points')

    def total_points(self):
        from july.game.models import Game, Player
        latest = Game.active_or_latest()
        kwargs = {
            self.rel_lookup: self,
            'game': latest
        }
        query = Player.objects.filter(**kwargs)
        total = query.aggregate(Sum('points'))
        points = total.get('points__sum')
        return points or 0

    @classmethod
    def create(cls, name):
        slug = slugify(name)
        if not slug:
            return None

        defaults = {
            'name': name,
            'approved': cls.auto_verify,
        }
        obj, created = cls.objects.get_or_create(slug=slug, defaults=defaults)

        if created and not cls.auto_verify:
            html = loader.render_to_string(cls.template, {'slug': slug})
            text = strip_tags(html)
            subject = "[group] %s awaiting approval." % slug
            mail_admins(subject, text, html_message=html)
        return obj


class Location(Group):
    """Simple model for holding point totals and projects for a location"""
    template = 'registration/location.html'
    rel_lookup = 'user__location'
    lookup = 'location'
    auto_verify = True

    def get_absolute_url(self):
        from django.core.urlresolvers import reverse
        return reverse('location-detail', kwargs={'slug': self.slug})


class Team(Group):
    """Simple model for holding point totals and projects for a Team"""
    template = 'registration/team.html'
    rel_lookup = 'user__team'
    lookup = 'team'
    auto_verify = False

    def get_absolute_url(self):
        from django.core.urlresolvers import reverse
        return reverse('team-detail', kwargs={'slug': self.slug})


class Language(models.Model):
    """Model for holding points and projects per programming language."""

    name = models.CharField(max_length=64)

    def __unicode__(self):
        return self.name


class UserBadge(models.Model):
    """Stores all badge info for a single user."""
    user = models.ForeignKey(settings.AUTH_USER_MODEL)
    badges = JSONField(blank=True, null=True)
