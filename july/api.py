
from collections import defaultdict
import json
import logging
import re
import urlparse
import requests
from os.path import splitext

from django.core.urlresolvers import reverse
from django import http
from django.template.defaultfilters import date
from django.views.generic.base import View
from django.views.decorators.csrf import csrf_exempt
from iso8601 import parse_date
from tastypie.resources import ModelResource
from tastypie.resources import ALL
from tastypie.resources import ALL_WITH_RELATIONS
from tastypie import fields

from july.people.models import Commit, Project, Location, Team, Language
from july.models import User

EMAIL_MATCH = re.compile('<(.+?)>')


class UserResource(ModelResource):

    class Meta:
        queryset = User.objects.all()
        excludes = ['password', 'email', 'is_superuser', 'is_staff',
                    'is_active']


class ProjectResource(ModelResource):

    class Meta:
        queryset = Project.objects.all()
        allowed_methods = ['get']
        filtering = {
            'user': ALL_WITH_RELATIONS,
            'locations': ALL,
            'teams': ALL
        }


class LocationResource(ModelResource):

    class Meta:
        queryset = Location.objects.all()
        allowed_methods = ['get']
        filtering = {
            'name': ['istartswith', 'exact', 'icontains'],
        }


class TeamResource(ModelResource):

    class Meta:
        queryset = Team.objects.all()
        allowed_methods = ['get']
        filtering = {
            'name': ['istartswith', 'exact', 'icontains'],
        }


class CommitResource(ModelResource):
    user = fields.ForeignKey(UserResource, 'user', blank=True, null=True)
    project = fields.ForeignKey(ProjectResource, 'project',
                                blank=True, null=True)

    class Meta:
        queryset = Commit.objects.all().select_related('user', 'project')
        allowed_methods = ['get']
        filtering = {
            'user': ALL_WITH_RELATIONS,
            'project': ALL_WITH_RELATIONS,
            'timestamp': ['exact', 'range', 'gt', 'lt'],
        }

    def gravatar(self, email):
        """Return a link to gravatar image."""
        url = 'http://www.gravatar.com/avatar/%s?s=48'
        from hashlib import md5
        email = email.strip().lower()
        hashed = md5(email).hexdigest()
        return url % hashed

    def dehydrate(self, bundle):
        email = bundle.data.pop('email')
        gravatar = self.gravatar(email)
        bundle.data['project_name'] = bundle.obj.project.name
        bundle.data['project_url'] = reverse('project-details',
                                             args=[bundle.obj.project.slug])
        bundle.data['username'] = getattr(bundle.obj.user, 'username', None)
        # Format the date properly using django template filter
        bundle.data['timestamp'] = date(bundle.obj.timestamp, 'c')
        bundle.data['picture_url'] = getattr(bundle.obj.user,
                                             'picture_url',
                                             gravatar)
        return bundle


class JSONMixin(object):

    def respond_json(self, data, **kwargs):
        content = json.dumps(data)
        resp = http.HttpResponse(content,
                                 content_type='application/json',
                                 **kwargs)
        resp['Access-Control-Allow-Origin'] = '*'
        return resp


class PlayerCommitsCollection(View, JSONMixin):

    def get(self):
        pass


def add_language(file_dict):
    """Parse a filename for the language.

    >>> d = {"file": "somefile.py", "type": "added"}
    >>> add_language(d)
    {"file": "somefile.py", "type": "added", "language": "Python"}
    """
    name = file_dict.get('file', '')
    language = None
    path, ext = splitext(name.lower())
    type_map = {
        #
        # C/C++
        #
        '.c': 'C/C++',
        '.cc': 'C/C++',
        '.cpp': 'C/C++',
        '.h': 'C/C++',
        '.hpp': 'C/C++',
        '.so': 'C/C++',
        #
        # C#
        #
        '.cs': 'C#',
        #
        # Clojure
        #
        '.clj': 'Clojure',
        #
        # Documentation
        #
        '.txt': 'Documentation',
        '.md': 'Documentation',
        '.rst': 'Documentation',
        '.hlp': 'Documentation',
        '.pdf': 'Documentation',
        '.man': 'Documentation',
        #
        # Erlang
        #
        '.erl': 'Erlang',
        #
        # Fortran
        #
        '.f': 'Fortran',
        '.f77': 'Fortran',
        #
        # Go
        #
        '.go': 'Golang',
        #
        # Groovy
        #
        '.groovy': 'Groovy',
        #
        # html/css/images
        #
        '.xml': 'html/css',
        '.html': 'html/css',
        '.htm': 'html/css',
        '.css': 'html/css',
        '.sass': 'html/css',
        '.less': 'html/css',
        '.scss': 'html/css',
        '.jpg': 'html/css',
        '.gif': 'html/css',
        '.png': 'html/css',
        '.jpeg': 'html/css',
        #
        # Java
        #
        '.class': 'Java',
        '.ear': 'Java',
        '.jar': 'Java',
        '.java': 'Java',
        '.war': 'Java',
        #
        # JavaScript
        #
        '.js': 'JavaScript',
        '.json': 'JavaScript',
        '.coffee': 'CoffeeScript',
        '.litcoffee': 'CoffeeScript',
        '.dart': 'Dart',
        #
        # Lisp
        #
        '.lisp': 'Common Lisp',
        #
        # Lua
        #
        '.lua': 'Lua',
        #
        # Objective-C
        #
        '.m': 'Objective-C',
        #
        # Perl
        #
        '.pl': 'Perl',
        #
        # PHP
        #
        '.php': 'PHP',
        #
        # Python
        #
        '.py': 'Python',
        '.pyc': 'Python',
        '.pyd': 'Python',
        '.pyo': 'Python',
        '.pyx': 'Python',
        '.pxd': 'Python',
        #
        # R
        #
        '.r': 'R',
        #
        # Ruby
        #
        '.rb': 'Ruby',
        #
        # Scala
        #
        '.scala': 'Scala',
        #
        # Scheme
        #
        '.scm': 'Scheme',
        '.scheme': 'Scheme',
        #
        # No Extension
        #
        '': '',
    }
    # Common extentionless files
    doc_map = {
        'license': 'Legalese',
        'copyright': 'Legalese',
        'changelog': 'Documentation',
        'contributing': 'Documentation',
        'readme': 'Documentation',
        'makefile': 'Build Tools',
    }
    if ext == '':
        language = doc_map.get(path)
    else:
        language = type_map.get(ext)
    file_dict['language'] = language
    return file_dict


def get_or_create_languages_list_from_files(files):
    result = []

    languages = [file['language'] for file in files]
    for language in set(languages):
        language_object, _ = Language.get_or_create(name=language)
        result.append(language_object)
    return result


class PostCallbackHandler(View, JSONMixin):

    def parse_commits(self, commits, project):
        """
        Takes a list of raw commit data and returns a dict of::

            {'email': [list of parsed commits]}

        """
        commit_dict = defaultdict(list)
        for k, v in [self._parse_commit(data, project) for data in commits]:
            # Did we not actual parse a commit?
            if v is None:
                continue
            commit_dict[k].append(v)

        return commit_dict

    def _parse_repo(self, repository):
        """Parse a repository."""
        raise NotImplementedError("Subclasses must define this")

    def _parse_commit(self, commit, project):
        """Parse a single commit."""
        raise NotImplementedError("Subclasses must define this")

    def parse_payload(self, request):
        """
        Hook for turning post data into payload.
        """
        payload = request.POST.get('payload')
        return payload

    def _publish_commits(self, commits):
        """Publish the commits to the real time channel."""
        host = self.request.META.get('HTTP_HOST', 'localhost:8000')
        url = 'http://%s/events/pub/' % host
        for commit in commits:
            try:
                resource = CommitResource()
                bundle = resource.build_bundle(obj=commit)
                # Make the timestamp a date object (again?)
                bundle.obj.timestamp = parse_date(bundle.obj.timestamp)
                dehydrated = resource.full_dehydrate(bundle)
                serialized = resource.serialize(
                    None, dehydrated, format='application/json')
                if commit.user:
                    requests.post(url + 'user-%s' % commit.user.id, serialized)
                requests.post(url + 'project-%s' % commit.project.id,
                              serialized)
                requests.post(url + 'global', serialized)
            except:
                logging.exception("Error publishing message")

    @csrf_exempt
    def dispatch(self, *args, **kwargs):
        return super(PostCallbackHandler, self).dispatch(*args, **kwargs)

    def post(self, request):
        payload = self.parse_payload(request)
        if not payload:
            return http.HttpResponseBadRequest()
        try:
            data = json.loads(payload)
        except:
            logging.exception("Unable to serialize POST")
            return http.HttpResponseBadRequest()

        commit_data = data.get('commits', [])

        repo = self._parse_repo(data)
        project = Project.create(**repo)

        commit_dict = self.parse_commits(commit_data, project)
        total_commits = []
        for email, commits in commit_dict.iteritems():
            # TODO: run this in a task queue?
            cmts = Commit.create_by_email(email, commits, project=project)
            total_commits += cmts

        status = 201 if len(total_commits) else 200

        self._publish_commits(total_commits)

        return self.respond_json(
            {'commits': [c.hash for c in total_commits]},
            status=status)


class BitbucketHandler(PostCallbackHandler):
    """
    Take a POST from bitbucket in the format::

        payload=>"{
            "canon_url": "https://bitbucket.org",
            "commits": [
                {
                    "author": "marcus",
                    "branch": "featureA",
                    "files": [
                        {
                            "file": "somefile.py",
                            "type": "modified"
                        }
                    ],
                    "message": "Added some featureA things",
                    "node": "d14d26a93fd2",
                    "parents": [
                        "1b458191f31a"
                    ],
                    "raw_author": "Marcus Bertrand <marcus@somedomain.com>",
                    "raw_node": "d14d26a93fd28d3166fa81c0cd3b6f339bb95bfe",
                    "revision": 3,
                    "size": -1,
                    "timestamp": "2012-05-30 06:07:03",
                    "utctimestamp": "2012-05-30 04:07:03+00:00"
                }
            ],
            "repository": {
                "absolute_url": "/marcus/project-x/",
                "fork": false,
                "is_private": true,
                "name": "Project X",
                "owner": "marcus",
                "scm": "hg",
                "slug": "project-x",
                "website": ""
            },
            "user": "marcus"
        }"
    """

    def _parse_repo(self, data):
        """Returns a dict suitable for creating a project.

        "repository": {
                "absolute_url": "/marcus/project-x/",
                "fork": false,
                "is_private": true,
                "name": "Project X",
                "owner": "marcus",
                "scm": "hg",
                "slug": "project-x",
                "website": ""
            }
        """
        if not isinstance(data, dict):
            raise AttributeError("Expected a dict object")

        repo = data.get('repository')
        canon_url = data.get('canon_url', '')

        abs_url = repo.get('absolute_url', '')
        if not abs_url.startswith('http'):
            abs_url = urlparse.urljoin(canon_url, abs_url)

        result = {
            'url': abs_url,
            'description': repo.get('website', ''),
            'name': repo.get('name'),
            'service': 'bitbucket'
        }

        fork = repo.get('fork', False)
        if fork:
            result['forked'] = True
        else:
            result['forked'] = False

        return result

    def _parse_email(self, raw_email):
        """
        Takes a raw email like: 'John Doe <joe@example.com>'

        and returns 'joe@example.com'
        """
        m = EMAIL_MATCH.search(raw_email)
        if m:
            return m.group(1)
        return ''

    @staticmethod
    def parse_extensions(data):
        """Returns a list of file extensions in the commit data"""
        file_dicts = data.get('files')
        extensions = [
            ext[1:] for root, ext in
            [splitext(file_dict['file']) for file_dict in file_dicts]]
        return extensions

    def _parse_commit(self, data, project):
        """Parse a single commit.

        Example::

            {
                "author": "marcus",
                "branch": "featureA",
                "files": [
                    {
                        "file": "somefile.py",
                        "type": "modified"
                    }
                ],
                "message": "Added some featureA things",
                "node": "d14d26a93fd2",
                "parents": [
                    "1b458191f31a"
                ],
                "raw_author": "Marcus Bertrand <marcus@somedomain.com>",
                "raw_node": "d14d26a93fd28d3166fa81c0cd3b6f339bb95bfe",
                "revision": 3,
                "size": -1,
                "timestamp": "2012-05-30 06:07:03",
                "utctimestamp": "2012-05-30 04:07:03+00:00"
            }
        """
        if not isinstance(data, dict):
            raise AttributeError("Expected a dict object")

        email = self._parse_email(data.get('raw_author'))
        files = map(add_language, data.get('files', []))
        languages = get_or_create_languages_list_from_files(files)
        project.languages.add(*languages)

        url = urlparse.urljoin(project.url, 'commits/%s' % data['raw_node'])

        commit_data = {
            'hash': data['raw_node'],
            'email': email,
            'author': data.get('author'),
            'name': data.get('author'),
            'message': data.get('message'),
            'timestamp': data.get('utctimestamp'),
            'url': data.get('url', url),
            'languages': languages,
            'files': files,
        }
        return email, commit_data


class GithubHandler(PostCallbackHandler):
    """
    Takes a POST response from github in the following format::

        payload=>"{
            "before": "5aef35982fb2d34e9d9d4502f6ede1072793222d",
            "repository": {
                "url": "http://github.com/defunkt/github",
                "name": "github",
                "description": "You're lookin' at it.",
                "watchers": 5,
                "forks": 2,
                "private": 1,
                "owner": {
                    "email": "chris@ozmm.org",
                    "name": "defunkt"
                }
            },
            "commits": [
            {
              "id": "41a212ee83ca127e3c8cf465891ab7216a705f59",
              "url": "http://github.com/defunkt/github/commit/41a212ef59",
              "author": {
                "email": "chris@ozmm.org",
                "name": "Chris Wanstrath"
              },
              "message": "okay i give in",
              "timestamp": "2008-02-15T14:57:17-08:00",
              "added": ["filepath.rb"]
            },
            {
              "id": "de8251ff97ee194a289832576287d6f8ad74e3d0",
              "url": "http://github.com/defunkt/github/commit/de8f8ae3d0",
              "author": {
                "email": "chris@ozmm.org",
                "name": "Chris Wanstrath"
              },
              "message": "update pricing a tad",
              "timestamp": "2008-02-15T14:36:34-08:00"
            }
            ],
            "after": "de8251ff97ee194a289832576287d6f8ad74e3d0",
            "ref": "refs/heads/master"
        }"
    """

    def _parse_repo(self, data):
        """Returns a dict suitable for creating a project."""
        if not isinstance(data, dict):
            raise AttributeError("Expected a dict object")

        data = data.get('repository')

        return {
            'url': data['url'],
            'description': data.get('description', ''),
            'name': data.get('name'),
            'forks': data.get('forks'),
            'watchers': data.get('watchers'),
            'service': 'github',
            'repo_id': data.get('id')
        }

    def _parse_files(self, data):
        """Make files look like bitbuckets json list."""
        def wrapper(key, data):
            return [{"file": f, "type": key} for f in data.get(key, [])]

        added = wrapper('added', data)
        modified = wrapper('modified', data)
        removed = wrapper('removed', data)
        return added + modified + removed

    def _parse_commit(self, data, project):
        """Return a tuple of (email, dict) to simplify commit creation.

        Raw commit data::

            {
              "id": "41a212ee83ca127e3c8cf465891ab7216a705f59",
              "url": "http://github.com/defunkt/github/commit/41a212ee83ca",
              "author": {
                "email": "chris@ozmm.org",
                "name": "Chris Wanstrath"
              },
              "message": "okay i give in",
              "timestamp": "2008-02-15T14:57:17-08:00",
              "added": ["filepath.rb"]
            },
        """
        if not isinstance(data, dict):
            raise AttributeError("Expected a dict object")

        author = data.get('author', {})
        email = author.get('email', '')
        name = author.get('name', '')
        files = map(add_language, self._parse_files(data))
        languages = get_or_create_languages_list_from_files(files)
        project.languages.add(*languages)

        commit_data = {
            'hash': data['id'],
            'url': data['url'],
            'email': email,
            'name': name,
            'message': data['message'],
            'timestamp': data['timestamp'],
            'languages': languages,
            'files': files,
        }
        return email, commit_data
