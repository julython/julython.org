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

            commit, created = cls.objects.get_or_create(
                hash=commit_hash,
                defaults=c)
            if created:
                created_commits.append(commit)

        return created_commits

    @classmethod
    def calendar(cls, end_date=None, days=35, **kwargs):
        """
        Returns number of commits per day for a date range.
        """
        if end_date and not end_date.tzinfo:
            end_date = end_date.replace(tzinfo=utc)
        else:
            end_date = end_date or datetime.utcnow().replace(tzinfo=utc)
        start_date = end_date - timedelta(days=days)
        count = cls.objects.filter(
            timestamp__range=(start_date, end_date), **kwargs) \
            .extra(select={'timestamp': 'date(timestamp)'}) \
            .values('timestamp').annotate(commit_count=Count('id'))
        return count


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
            try:
                project = cls.objects.get(service=service, repo_id=repo_id)
                created = False
            except cls.DoesNotExist:
                project, created = cls.objects.get_or_create(
                    slug=slug, defaults=kwargs)

        if not project.active:
            return None
        # Update stale project information.
        project.update(slug, **kwargs)
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
        return kwargs

    def update(self, slug, **kwargs):
        last = now() - self.updated_on
        if last.seconds >= (3600 * 6):
            for key, value in self._get_bitbucket_data(**kwargs):
                setattr(self, key, value)
            self.slug = slug
            self.save()

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
    slug = models.SlugField(primary_key=True)
    name = models.CharField(max_length=64, blank=False)
    total = models.IntegerField(default=0)
    approved = models.BooleanField(default=False)

    class Meta:
        abstract = True

    def __str__(self):
        return self.name

    def __unicode__(self):
        return self.name

    def members_by_points(self):
        raise NotImplementedError("members_by_points must be implemented "
                                  "by the subclass!")

    def total_points(self):
        raise NotImplementedError("total_points must be implemented "
                                  "by the subclass!")

    @classmethod
    def create(cls, slug, name):
        slug = slugify(name)
        if not slug:
            return None
        obj, created = cls.objects.get_or_create(slug=slug,
                                                 defaults={'name': name})
        if created:
            html = loader.render_to_string(cls.template, {'slug': slug})
            text = strip_tags(html)
            subject = "[group] %s awaiting approval." % slug
            mail_admins(subject, text, html_message=html)
        return obj


class Location(Group):
    """Simple model for holding point totals and projects for a location"""
    template = 'registration/location.html'

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
        return reverse('location-detail', kwargs={'slug': self.slug})


class Team(Group):
    """Simple model for holding point totals and projects for a Team"""
    template = 'registration/team.html'

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


class Language(models.Model):
    """Model for holding points and projects per programming language."""

    name = models.CharField(max_length=64)

    def __unicode__(self):
        return self.name
