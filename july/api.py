
from collections import defaultdict
import json
import logging
import re
import urlparse

from django import http
from django.views.generic.base import View
from django.views.decorators.csrf import csrf_exempt
from iso8601 import parse_date
from tastypie.resources import ModelResource
from tastypie.resources import ALL
from tastypie.resources import ALL_WITH_RELATIONS
from tastypie import fields

from july.people.models import Commit, Project
from july.models import User

EMAIL_MATCH = re.compile('<(.+?)>')

class UserResource(ModelResource):
    
    class Meta:
        queryset = User.objects.all()
        excludes = ['password', 'email', 'is_superuser']


class ProjectResource(ModelResource):
    
    class Meta:
        queryset = Project.objects.all()
        allowed_methods = ['get']
        filtering = {
            'user': ALL_WITH_RELATIONS,
            'locations': ALL,
            'teams': ALL 
        }


class CommitResource(ModelResource):
    user = fields.ForeignKey(UserResource, 'user', blank=True, null=True)
    project = fields.ForeignKey(ProjectResource, 'project', blank=True, null=True)
    
    class Meta:
        queryset = Commit.objects.all().select_related().order_by('-timestamp')
        allowed_methods = ['get']
        default_format = 'application/json'
        filtering = {
            'user': ALL_WITH_RELATIONS,
            'project': ALL_WITH_RELATIONS,
            'timestamp': ['exact', 'range', 'gt', 'lt'],
        }
    
    def dehydrate(self, bundle):
        #logging.error(dir(bundle))
        #logging.error(bundle.obj.user)
        bundle.data['username'] = getattr(bundle.obj.user, 'username', '')
        bundle.data['picture_url'] = getattr(bundle.obj.user, 'picture_url', '')
            
        #logging.error(bundle.data)
        return bundle


class JSONMixin(object):
    
    def respond_json(self, data, **kwargs):
        content = json.dumps(data)
        resp = http.HttpResponse(content, content_type='application/json', **kwargs)
        resp['Access-Control-Allow-Origin'] = '*'
        return resp

class PlayerCommitsCollection(View, JSONMixin):
    
    def get(self):
        pass

class PostCallbackHandler(View, JSONMixin):

    def parse_commits(self, commits):
        """
        Takes a list of raw commit data and returns a dict of::
            
            {'email': [list of parsed commits]}
        
        """
        commit_dict = defaultdict(list)
        for k, v in [self._parse_commit(data) for data in commits]:
            # Did we not actual parse a commit?
            if v is None:
                continue
            commit_dict[k].append(v)
        
        return commit_dict
    
    def _parse_repo(self, repository):
        """Parse a repository."""
        raise NotImplementedError("Subclasses must define this")
    
    def _parse_commit(self, commit):
        """Parse a single commit."""
        raise NotImplementedError("Subclasses must define this")
    
    def parse_payload(self, request):
        """
        Hook for turning post data into payload.
        """
        payload = request.POST.get('payload')
        return payload
    
    
    @csrf_exempt
    def dispatch(self, *args, **kwargs):
        return super(PostCallbackHandler, self).dispatch(*args, **kwargs)
    
    def post(self, request):
        payload = self.parse_payload(request)
        logging.info(payload)
        if not payload:
            raise http.HttpResponseBadRequest
        
        try:
            data = json.loads(payload)
        except:
            logging.exception("Unable to serialize POST")
            raise http.HttpResponseBadRequest
        
        commit_data = data.get('commits', [])
        
        repo = self._parse_repo(data)
        project, _ = Project.create(**repo)
        
        commit_dict = self.parse_commits(commit_data)
        total_commits = []
        for email, commits in commit_dict.iteritems():
            # TODO: run this in a task queue?
            cmts = Commit.create_by_email(email, commits, project=project)
            total_commits += cmts
        
        status = 201 if len(total_commits) else 200
        
        return self.respond_json({'commits': [c.hash for c in total_commits]}, status=status)
        
    
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
        
    def _parse_commit(self, data):
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
        
        commit_data = {
            'hash': data['raw_node'],
            'email': email,
            'author': data.get('author'),
            'name': data.get('author'),
            'message': data.get('message'),
            'timestamp': data.get('utctimestamp'),
            'url': data.get('url', ''),
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
                  "url": "http://github.com/defunkt/github/commit/41a212ee83ca127e3c8cf465891ab7216a705f59",
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
                  "url": "http://github.com/defunkt/github/commit/de8251ff97ee194a289832576287d6f8ad74e3d0",
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
            'watchers': data.get('watchers')
        }
    
    
    def _parse_commit(self, data):
        """Return a tuple of (email, dict) to simplify commit creation.
        
        Raw commit data::
        
            {
              "id": "41a212ee83ca127e3c8cf465891ab7216a705f59",
              "url": "http://github.com/defunkt/github/commit/41a212ee83ca127e3c8cf465891ab7216a705f59",
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
        
        commit_data = {
            'hash': data['id'],
            'url': data['url'],
            'email': email,
            'name': name,
            'message': data['message'],
            'timestamp': data['timestamp'],
        }
        return email, commit_data
