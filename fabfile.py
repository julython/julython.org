"""
Common tools for deploy run shell and the like.
"""

import os
from string import Template
from urllib import urlencode

from fabric.api import lcd, task, local, put, env


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
def load(email=None):
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
        with open(json_file) as post:
            p = Template(post.read()).substitute({'__EMAIL__': email})
            payload = urlencode({'payload': p})
            local('curl http://localhost:8000/api/v1/github -s -d %s' % payload)

    for json_file in bitbucket:
        with open(json_file) as post:
            p = Template(post.read()).substitute({'__EMAIL__': email})
            payload = urlencode({'payload': p})
            local('curl http://localhost:8000/api/v1/bitbucket -s -d %s' % payload)


@task
def install():
    """Install the node_modules dependencies"""
    local('git submodule update --init')
    with lcd('assets'):
        local('npm install')


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
    local("tar -czvf july.tar.gz"
          " --exclude '\.*'"
          " --exclude '*.pyc'"
          " --exclude 'assets*'"
          " --exclude 'htmlcov*'"
          " --exclude '*.db'"
          " --exclude '*.tar.gz'"
          " *")
    put('july.tar.gz', '/tmp/')
