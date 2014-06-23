"""
Common tools for deploy run shell and the like.
"""

import os
from string import Template
from urllib import urlencode

from fabric.api import *
from fabric.colors import *
from fabric.contrib.project import rsync_project
import requests

@task
def pep8():
    """Run Pep8"""
    local("pep8 july --exclude='*migrations*','*static*'")


@task
def test(coverage='False', skip_js='False'):
    """Run the test suite"""
    if coverage != 'False':
        local("rm -rf htmlcov")
        local("coverage run --include='july*' --omit='*migration*' manage.py test")
        local("coverage html")
    else:
        local("python manage.py test july people game")
    if skip_js == 'False':
        with lcd('assets'):
            local('node_modules/grunt-cli/bin/grunt jasmine')
    pep8()


@task
def load(email=None, port=8000):
    """Manually send a POST to api endpoints."""
    if not email:
        print "You must provide an email address 'fab load:me@foo.com'"
        return

    github = []
    bitbucket = []
    for json_file in os.listdir('data'):
        if json_file.startswith('github'):
            github.append(os.path.join('data', json_file))
        elif json_file.startswith('bitbucket'):
            bitbucket.append(os.path.join('data', json_file))

    for json_file in github:
        with open(json_file) as post_file:
            post = Template(post_file.read()).substitute({'__EMAIL__': email})
            payload = {'payload': post}
            url = 'http://localhost:%s/api/v1/github' % port
            response = requests.post(url, data=payload)
            print(cyan("%s %s" % (response.status_code, response.reason)))
            response.raise_for_status()

    for json_file in bitbucket:
        with open(json_file) as post_file:
            post = Template(post_file.read()).substitute({'__EMAIL__': email})
            payload = {'payload': post}
            url = 'http://localhost:%s/api/v1/bitbucket' % port
            response = requests.post(url, data=payload)
            print(cyan("%s %s" % (response.status_code, response.reason)))
            response.raise_for_status()


@task
def install():
    """Install the node_modules dependencies"""
    local('git submodule update --init')
    if not os.path.isfile('july/secrets.py'):
        local('cp july/secrets.py.template july/secrets.py')

    local('python manage.py syncdb')
    local('python manage.py migrate')
    local('python manage.py loaddata july/fixtures/development.json')

    with lcd('assets'), settings(warn_only=True):
        out = local('npm install')
        if out.failed:
            print(red("Problem running npm, did you install node?"))


@task
def watch():
    """Grunt watch development files"""
    with lcd('assets'):
        local('node_modules/grunt-cli/bin/grunt concat less:dev watch')


@task
def compile():
    """Compile assets for production."""
    with lcd('assets'):
        local('node_modules/grunt-cli/bin/grunt less:prod uglify')


@task
def staging(user='rmyers'):
    env.hosts = ['january.julython.org']
    env.user = user


@task
def deploy():
    """Deploy to production"""
    compile()
    exclude = ['*.pyc', '*.db', 'htmlcov*', '.git', 'assets/node_modules']
    rsync_project('/tmp', exclude=exclude)
