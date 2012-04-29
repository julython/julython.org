"""
Common tools for deploy run shell and the like.
"""
import os
import sys
import code

from fabric import *

import google

os.environ['DJANGO_SETTINGS_MODULE'] = 'july.settings'

DIR_PATH = os.path.abspath(os.path.dirname(os.path.dirname(google.__file__)))

EXTRA_PATHS = [
  os.getcwd(),
  DIR_PATH,
  os.path.join(DIR_PATH, 'lib', 'antlr3'),
  os.path.join(DIR_PATH, 'lib', 'django_1_3'),
  os.path.join(DIR_PATH, 'lib', 'fancy_urllib'),
  os.path.join(DIR_PATH, 'lib', 'ipaddr'),
  os.path.join(DIR_PATH, 'lib', 'jinja2'),
  os.path.join(DIR_PATH, 'lib', 'protorpc'),
  os.path.join(DIR_PATH, 'lib', 'markupsafe'),
  os.path.join(DIR_PATH, 'lib', 'webob_0_9'),
  os.path.join(DIR_PATH, 'lib', 'webapp2'),
  os.path.join(DIR_PATH, 'lib', 'yaml', 'lib'),
  os.path.join(DIR_PATH, 'lib', 'simplejson'),
  os.path.join(DIR_PATH, 'lib', 'google.appengine._internal.graphy'),
]

def _setup_paths():
    """Setup sys.path with everything we need to run."""
    sys.path = EXTRA_PATHS + sys.path

def shell(remote=0, clear_datastore=0):
    """
    Run an interactive shell with the datastore. 
    """
    
    _setup_paths()
    
    from google.appengine.tools import dev_appserver, dev_appserver_main
    from google.appengine.ext import db, deferred
    from google.appengine.api import memcache, urlfetch
    
    appinfo, _, _ = dev_appserver.LoadAppConfig(
        os.getcwd(), {}, default_partition='')
    
    # TODO: pass in args form fabric
    _, opts = dev_appserver_main.ParseArguments([])
    
    dev_appserver.SetupStubs(appinfo.application, **opts)
    # Build a dict usable as locals() from the modules we want to use
    modname = lambda m: m.__name__.rpartition('.')[-1]
    mods = [db, deferred, memcache, sys, urlfetch]
    mods = dict((modname(m), m) for m in mods)

    # The banner for either kind of shell
    banner = 'Python %s\n\nImported modules: %s\n' % (
        sys.version, ', '.join(sorted(mods)))
    
    sys.ps1 = '>>> '
    sys.ps2 = '... '
    code.interact(banner=banner, local=mods)

def remote_shell():
    """Run a remote shell."""
    
    _setup_paths()
    
    from google.appengine.tools import remote_api_shell
    from google.appengine.tools import appengine_rpc
    
    remote_api_shell.remote_api_shell('julython.appspot.com', 's~julython', 
        remote_api_shell.DEFAULT_PATH, True, appengine_rpc.HttpRpcServer)