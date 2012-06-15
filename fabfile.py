"""
Common tools for deploy run shell and the like.
"""
from fabric.api import lcd

from gae_django.fabric_commands import *

@task
def less_compile(compress=True):
    """Compile the .less files into css."""
    with lcd('july'):
        local("lessc --compress static_root/less/layout.less > static_root/css/main.css")