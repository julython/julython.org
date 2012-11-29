"""
Common tools for deploy run shell and the like.
"""
from fabric.api import lcd, task, local

#from gae_django.fabric_commands import local_shell, remote_shell, runserver
#from gae_django.fabric_commands import deploy as _deploy


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
        local('node_modules/grunt/bin/grunt less:prod min')


@task
def deploy(version='', appid='.'):
    """Deploy to production"""
    compile()
    _deploy(version, appid)
