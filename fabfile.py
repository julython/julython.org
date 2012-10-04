"""
Common tools for deploy run shell and the like.
"""
from fabric.api import lcd

from gae_django.fabric_commands import *

@task
def test():
    """Run the test suite"""
    local("python -m unittest discover")
    with lcd('assets'):
        local('node_modules/grunt/bin/grunt jasmine')

@task
def install():
    """Install the node_modules dependencies"""
    with lcd('assets'):
        local('npm install')

@task
def watch():
    """Grunt watch development files"""
    with lcd('assets'):
        local('node_modules/grunt/bin/grunt concat less:dev watch')

@task
def compile():
    """Compile assets for production."""
    with lcd('assets'):
        local('node_modules/grunt/bin/grunt less:dev concat watch')