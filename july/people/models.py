import logging
from urlparse import urlparse

from django.db import models, transaction
from django.conf import settings


class Commit(models.Model):
    """
    Commit record for the profile, the parent is the profile
    that way we can update the commit count and last commit timestamp
    in the same transaction.
    """
    user = models.ForeignKey(settings.AUTH_USER_MODEL, blank=True, null=True)
    hash = models.CharField(max_length=255)
    author = models.CharField(max_length=255)
    name = models.CharField(max_length=255)
    email = models.CharField(max_length=255)
    message = models.CharField(max_length=255)
    url = models.CharField(max_length=255)
    project = models.ForeignKey("Project")
    timestamp = models.DateTimeField()
    created_on = models.DateTimeField(auto_now_add=True)
    
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
    
    @transaction.commit_on_success
    @classmethod
    def create_by_user(cls, user, commits, project=None):
        """Create a commit with parent user, updating users points."""
        created_commits = []
        
        for c in commits:
            c['user'] = user
            c['project'] = project
            commit_hash = c.pop('hash', None)
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
                user.projects.add(project)
                user.save()
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
    total = models.IntegerField(default=0)
    
    def __str__(self):
        if self.name:
            return self.name
        else:
            return self.url
    
    def __unicode__(self):
        return self.__str__()
    
    @property
    def project_name(self):
        return self.parse_project_name(self.url)
    
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
        return '%s-%s' % (host_abbr, name)


class Group(models.Model):
    total = models.IntegerField(default=0)
    name = models.CharField(max_length=64, blank=False)
    slug = models.SlugField(blank=False)

    class __Meta__:
        abstract=True

    def __str__(self):
        return self.name

    def __unicode__(self):
        return self.name


class Location(Group):
    """Simple model for holding point totals and projects for a location"""

    def get_absolute_url(self):
        from django.core.urlresolvers import reverse
        return reverse('member-list', kwargs={'location_slug': self.slug})


class Team(Group):
    """Simple model for holding point totals and projects for a Team"""

    def get_absolute_url(self):
        from django.core.urlresolvers import reverse
        return reverse('team-details', kwargs={'team_slug': self.slug})
