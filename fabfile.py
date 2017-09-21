"""
Common tools for deploy run shell and the like.
"""

import os
from string import Template
from urllib import urlencode

from fabric.api import *
from fabric.colors import *
from fabric.contrib.project import rsync_project
from fabric.contrib.files import exists
import requests

env.use_ssh_config = True


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
def load(email=None, port=80):
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
            url = 'http://127.0.0.1:%s/api/v1/github' % port
            response = requests.post(url, data=payload)
            print(cyan("%s %s" % (response.status_code, response.reason)))
            response.raise_for_status()

    for json_file in bitbucket:
        with open(json_file) as post_file:
            post = Template(post_file.read()).substitute({'__EMAIL__': email})
            payload = {'payload': post}
            url = 'http://127.0.0.1:%s/api/v1/bitbucket' % port
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
def staging(user='julython'):
    env.hosts = ['january.julython.org']
    env.user = user


@task
def prod(user='julython'):
    env.hosts = ['july.julython.org']
    env.user = user


@task
def bootstrap(user='julython'):
    """Install packages and docker on a new host."""
    sudo('apt-get install -y python-pip python-dev software-properties-common')
    sudo('apt-get install -y apt-transport-https ca-certificates curl')
    run('curl -fsSL https://download.docker.com/linux/ubuntu/gpg |'
        ' sudo apt-key add -')
    sudo('add-apt-repository "deb [arch=amd64]'
         ' https://download.docker.com/linux/ubuntu'
         ' $(lsb_release -cs) stable"')
    sudo('apt-get update')
    sudo('apt-get install -y docker-ce')
    sudo('groupadd docker')
    sudo('usermod -aG docker {user}'.format(user=user))
    sudo('pip install docker-compose')


@task
def deploy(migrate='no'):
    """Deploy to production"""
    compile()
    local('python manage.py collectstatic')
    exclude = ['*.pyc', '*.db', 'htmlcov*', '.git', 'assets/*', 'data']
    sudo('mkdir -p /usr/local/julython.org')
    sudo('chown julython:julython /usr/local/julython.org')
    rsync_project('/usr/local', exclude=exclude)

    with cd('/usr/local/julython.org'):
        run('make clean')
        run('make build')
        run('make prod')

    if migrate is not 'no':
        print(red("Please migrate by hand!"))
